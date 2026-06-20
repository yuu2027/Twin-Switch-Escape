using System;
using UnityEngine;

namespace TwinSwitchEscape
{
    public enum GameStatus
    {
        Playing,
        Cleared,
        Failed
    }

    /// <summary>
    /// Phase 1 のローカル権威。鍵・スイッチ・出口の状態を集計し、
    /// 出口開放／クリア／失敗を判定する。
    ///
    /// 各ギミック（KeyPickup / PressSwitch / ExitDoor）は自分の状態を公開するだけで、
    /// 集計と判定は本クラスがポーリングして行う（疎結合）。
    /// 将来 Go サーバー権威へ移す際は、この判定をサーバーへ移植する想定。
    /// </summary>
    public class GameManager : MonoBehaviour
    {
        [Tooltip("制限時間（秒）。spec の timeLimitSec=180 が既定。")]
        [SerializeField] private float timeLimitSec = 180f;

        [Header("ギミック参照")]
        [SerializeField] private KeyPickup[] keys;
        [SerializeField] private PressSwitch[] switches;
        [SerializeField] private ExitDoor exitDoor;

        public GameStatus Status { get; private set; } = GameStatus.Playing;
        public float RemainingTime { get; private set; }
        public int TotalKeyCount => keys != null ? keys.Length : 0;
        public int CollectedKeyCount { get; private set; }
        public bool ExitOpen => exitDoor != null && exitDoor.IsOpen;

        /// <summary>状態に意味のある変化があったときに発火（HUD 更新用）。</summary>
        public event Action StateChanged;

        private int lastWholeSecond = -1;

        private void Start()
        {
            RemainingTime = timeLimitSec;
            CollectedKeyCount = CountCollectedKeys();
            RaiseStateChanged();
        }

        private void Update()
        {
            if (Status != GameStatus.Playing)
            {
                return;
            }

            TickTimer();
            if (Status != GameStatus.Playing)
            {
                return;
            }

            UpdateKeyCount();
            UpdateExitOpen();
            CheckClear();
        }

        private void TickTimer()
        {
            RemainingTime -= Time.deltaTime;
            if (RemainingTime <= 0f)
            {
                RemainingTime = 0f;
                SetStatus(GameStatus.Failed);
                return;
            }
            // CeilToInt
            int whole = Mathf.CeilToInt(RemainingTime);
            if (whole != lastWholeSecond)
            {
                lastWholeSecond = whole;
                RaiseStateChanged();
            }
        }

        private void UpdateKeyCount()
        {
            int collected = CountCollectedKeys();
            if (collected != CollectedKeyCount)
            {
                CollectedKeyCount = collected;
                RaiseStateChanged();
            }
        }

        private void UpdateExitOpen()
        {
            if (exitDoor == null)
            {
                return;
            }

            bool allKeys = TotalKeyCount > 0 && CollectedKeyCount >= TotalKeyCount;
            if (allKeys && AllSwitchesPressed())
            {
                exitDoor.SetOpen(true);
                RaiseStateChanged();
            }
            if (!AllSwitchesPressed())
            {
                exitDoor.SetOpen(false);
            }
        }

        private void CheckClear()
        {
            if (exitDoor != null && exitDoor.IsOpen && exitDoor.PlayerInside)
            {
                SetStatus(GameStatus.Cleared);
            }
        }

        // プレイヤーが入手しているキーの数
        private int CountCollectedKeys()
        {
            if (keys == null)
            {
                return 0;
            }
            int count = 0;
            foreach (var key in keys)
            {
                if (key != null && key.Collected)
                {
                    count++;
                }
            }
            return count;
        }

        private bool AllSwitchesPressed()
        {
            if (switches == null || switches.Length == 0)
            {
                return false;
            }
            foreach (var sw in switches)
            {
                if (sw == null || !sw.IsPressed)
                {
                    return false;
                }
            }
            return true;
        }

        private void SetStatus(GameStatus status)
        {
            if (Status == status)
            {
                return;
            }
            Status = status;
            RaiseStateChanged();
        }

        private void RaiseStateChanged()
        {
            StateChanged?.Invoke();
        }
    }
}
