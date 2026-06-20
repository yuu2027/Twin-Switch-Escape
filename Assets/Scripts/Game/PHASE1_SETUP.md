# Phase 1 セットアップ手順

スクリプトは `Assets/Scripts/Game/` に用意済み（すべて名前空間 `TwinSwitchEscape`）。
以下を Unity Editor で実施すると、ローカルでクリア可能な状態になる。

> 用語: 「トリガにする」= 対象の Collider2D の **Is Trigger** にチェック。

---

## 0. Input System を有効化（最初に一度だけ）

`ProjectSettings/ProjectSettings.asset` の `activeInputHandler` は **2（Both）** に変更済み。
反映には **Unity の再起動が必要**。再起動後、コンパイルエラーが無いことを確認する。

（Edit > Project Settings > Player > Active Input Handling が「Both」になっていれば OK）

---

## 1. マップと壁の当たり判定

1. `GameScene` を開く。
2. `Assets/Grid/` の **GrassTilemap / StoneTilemap / WallTulemap** をシーンに配置（ヒエラルキーへドラッグ）。
   - 床（Grass/Stone）と壁（Wall）の位置が重なるように合わせる。
3. **壁に当たり判定を付ける**（重要：現状コライダー無し）。
   `WallTulemap` 内の Tilemap を持つ子オブジェクト（`Layer1`）を選択し、Add Component で次の3つを追加：
   - **Tilemap Collider 2D**
   - **Composite Collider 2D**（追加すると Rigidbody 2D も自動追加される）
   - 追加された **Rigidbody 2D** の **Body Type = Static** にする
   - **Tilemap Collider 2D** の **Used By Composite** にチェック
   - （任意）Composite Collider 2D の Geometry Type = Polygons
   → これでプレイヤーが壁で止まる。

---

## 2. LocalPlayer（単色の仮プレイヤー）

1. 仮スプライトを用意：Project で右クリック → Create → 2D → Sprites → **Square**（白い四角）。
2. 空の GameObject を作り名前を **LocalPlayer**、Tag を Player（任意）に。
3. 以下を追加・設定：
   - **Sprite Renderer**：Sprite に手順1の Square を割当。Sorting Layer / Order はプレイヤーが床より前に出るよう調整（Order を大きめに）。
   - **Rigidbody 2D**：Gravity Scale = 0、Constraints の **Freeze Rotation Z** にチェック
     （※`PlayerController` が起動時に自動でも設定するが、明示推奨）。
   - **Box Collider 2D**：体の大きさに合わせる。
   - **Player Input**（Input System）：
     - Actions = `Assets/InputSystem_Actions`
     - Default Map = **Player**
     - Behavior = **Send Messages**
   - **PlayerController**（自作）：Move Speed を 4 程度に。
4. プレイヤーを壁の内側のスタート地点に置く。

> アニメは後付け。今は付けない。→ 後述「アニメーション追加」参照。

---

## 3. 鍵（Key_1 / Key_2）

1. `Assets/Arts/Prefabs/Items/key` をシーンに2つ配置し、`Key_1` / `Key_2` にリネーム。
2. 各キーに **Collider2D**（Circle か Box）を追加し、**Is Trigger** にチェック（取得範囲）。
3. **KeyPickup**（自作）を追加。Key Id に `key_1` / `key_2` を設定。Sprite Renderer は自動取得。
4. プレイヤーが鍵に重なって **E** を押すと取得 → 鍵が消える。

---

## 4. スイッチ（Switch_A / Switch_B）

1. `Assets/Arts/Prefabs/Items/Chest_Closed` を2つ配置し、`Switch_A` / `Switch_B` にリネーム。
2. **Collider2D** を追加して **Is Trigger** にチェック（踏み判定の範囲）。
3. **PressSwitch**（自作）を追加：
   - Switch Id = `switch_a` / `switch_b`
   - Hold Seconds = 3（離れても3秒 ON を保持＝1人で両方踏みに行ける猶予）
   - （任意）Sprite Renderer / Pressed Sprite(Chest_Open の Sprite) / Released Sprite(Chest_Closed の Sprite) を割当てると ON/OFF で見た目が変わる。

---

## 5. 出口（Exit）

1. `Assets/Arts/Prefabs/Items/Door_Closed` を配置し、`Exit` にリネーム。
2. **Collider2D** を追加して **Is Trigger** にチェック（出口範囲）。
3. **ExitDoor**（自作）を追加：
   - （任意）Sprite Renderer / Open Sprite(Door_Open の Sprite) / Closed Sprite(Door_Closed の Sprite) を割当て。
   - 開くと Open Sprite に切り替わる。

---

## 6. Managers（GameManager）

1. 空の GameObject **Managers** を作成し、**GameManager**（自作）を追加。
2. Inspector で結線：
   - Time Limit Sec = 180
   - **Keys**：要素2 = Key_1, Key_2
   - **Switches**：要素2 = Switch_A, Switch_B
   - **Exit Door** = Exit
3. 判定はすべて GameManager が毎フレーム集計（鍵・スイッチ参照を入れるだけ）。

---

## 7. UI（GameHUD）

1. ヒエラルキーで UI > Canvas を作成（Render Mode = Screen Space - Overlay）。
2. Canvas 配下に **Text - TextMeshPro** を2つ作成：
   - `TimerText`（残り時間と鍵カウント表示）
   - `StatusText`（CLEAR! / FAILED / EXIT OPEN! 表示。中央・大きめ推奨）
   - 初回は TMP Essentials のインポートを求められたらインポートする。
3. Canvas（または任意のオブジェクト）に **GameHUD**（自作）を追加し結線：
   - Game Manager = Managers
   - Timer Text = TimerText
   - Status Text = StatusText

---

## 8. カメラ追従（既存スクリプト再利用）

1. Main Camera に **CameraFollow**（`Cainos.PixelArtTopDown_Basic.CameraFollow`）を追加。
2. Target = LocalPlayer の Transform。

---

## 9. 動作確認（Play）

- WASD／矢印で移動でき、壁で止まる（Input System 経由）。
- 鍵に重なって **E** で取得 → HUD の Keys が増える。
- 片方のスイッチを踏んで離れ、3秒以内にもう片方を踏むと両 ON 成立 → 出口が開く（Door_Open）。
- 全鍵取得かつ出口が開いた状態で出口に入ると **CLEAR!**。
- 制限時間 0 で **FAILED**。

---

## アニメーション追加（後付け）

ロジック（`PlayerController`）と表示は分離済み。`Assets/Arts/Animations/` に
`player_01.controller` と 5 クリップ（Player01_Idle / _Down / _Up / _Left / _Right）が作成済み。

> `player_01.controller` は下記のパラメータ・ステート・遷移で **設定済み**（Any State 構成）。
> GUI で組み直す必要はない。LocalPlayer に Animator + PlayerAnimationBridge を付けるだけ。

### コンポーネント
LocalPlayer に以下を追加：
1. **Animator**：Controller = `player_01.controller`。
2. **PlayerAnimationBridge**（自作）：Animator を割当て（未割当なら同オブジェクトから自動取得）。
   - `PlayerController` の向き・移動状態を Animator パラメータへ反映するだけ。ロジックは変更不要。

### パラメータ（Animator の Parameters タブ）
| 名前 | 型 | 既定値 | 用途 |
|---|---|---|---|
| `IsMoving` | **Bool** | false | 移動中かどうか（Idle ⇔ 移動 の切替）※新規追加 |
| `Direction` | **Int** | 0 | 向き。0=Down / 1=Up / 2=Left / 3=Right ※作成済み |

> 既存の空ステート「Locommotion」と Blend Tree は削除してよい。

### ステート（各クリップを1ステートに）
- **Idle** … Player01_Idle（**Default State** にする）
- **MoveDown** … Player01_Down
- **MoveUp** … Player01_Up
- **MoveLeft** … Player01_Left
- **MoveRight** … Player01_Right

### 遷移表（Any State を起点にすると最小構成）
すべての遷移で **Has Exit Time = OFF**、**Transition Duration = 0**、
Any State 発の遷移は **Can Transition To Self = OFF** にする。

| From | To | 条件 |
|---|---|---|
| Any State | Idle | `IsMoving` = false |
| Any State | MoveDown | `IsMoving` = true かつ `Direction` Equals 0 |
| Any State | MoveUp | `IsMoving` = true かつ `Direction` Equals 1 |
| Any State | MoveLeft | `IsMoving` = true かつ `Direction` Equals 2 |
| Any State | MoveRight | `IsMoving` = true かつ `Direction` Equals 3 |

> Any State を使わない場合は、Idle→各Move（IsMoving=true & Direction==n）、
> 各Move→Idle（IsMoving=false）、Move同士（Direction==別の値）を張っても同じ挙動になる。

> Cainos の `PF Player` プレハブをそのまま使う場合は、付属の旧 `TopDownCharacterController`（旧 Input 依存）を
> 外し、本 Phase1 の `PlayerController` + `Player Input` + `PlayerAnimationBridge` に差し替えること。
