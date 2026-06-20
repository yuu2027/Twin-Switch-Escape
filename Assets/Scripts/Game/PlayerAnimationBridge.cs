using UnityEngine;

namespace TwinSwitchEscape
{
    /// <summary>
    /// アニメーション後付け用の薄い橋渡しコンポーネント。
    ///
    /// PlayerController（ロジック）から MoveInput / Facing を読み取り、
    /// Animator のパラメータへ反映するだけ。ゲームロジックには一切手を入れない。
    ///
    /// Animator には2つのパラメータを設定する：
    ///   IsMoving : Bool  … 移動中かどうか（Idle ⇔ 移動 の切替）
    ///   Direction: Int   … 向き。0=Down, 1=Up, 2=Left, 3=Right
    /// 状態（クリップ）は Idle / Down / Up / Left / Right の5つ。
    /// </summary>
    [RequireComponent(typeof(PlayerController))]
    public class PlayerAnimationBridge : MonoBehaviour
    {
        [SerializeField] private Animator animator;
        [SerializeField] private string directionParam = "Direction";
        [SerializeField] private string isMovingParam = "IsMoving";

        private PlayerController controller;

        private void Awake()
        {
            controller = GetComponent<PlayerController>();
            if (animator == null)
            {
                animator = GetComponent<Animator>();
            }

            if (animator == null)
            {
                Debug.LogError("[PlayerAnimationBridge] Animator が見つかりません。", this);
            }
        }

        private void Update()
        {
            if (animator == null || controller == null)
            {
                return;
            }

            Vector2 move = controller.MoveInput;
            Vector2 facing = controller.Facing;
            bool isMoving = move.sqrMagnitude > 0.01f;
            int direction = DirectionToInt(facing);

            animator.SetBool(isMovingParam, isMoving);
            animator.SetInteger(directionParam, direction);
        }

        private static int DirectionToInt(Vector2 facing)
        {
            // 0=Down, 1=Up, 2=Left, 3=Right
            if (Mathf.Abs(facing.x) > Mathf.Abs(facing.y))
            {
                return facing.x >= 0f ? 3 : 2; // Right : Left
            }
            return facing.y >= 0f ? 1 : 0;     // Up : Down
        }
    }
}
