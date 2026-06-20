using TMPro;
using UnityEngine;

namespace TwinSwitchEscape
{
    /// <summary>
    /// 残り時間・鍵カウント・ゲーム状態を画面に表示する。
    /// GameManager の StateChanged を購読して更新する。
    /// </summary>
    public class GameHUD : MonoBehaviour
    {
        [SerializeField] private GameManager gameManager;
        [SerializeField] private TMP_Text timerText;
        [SerializeField] private TMP_Text statusText;

        private void OnEnable()
        {
            if (gameManager != null)
            {
                gameManager.StateChanged += Refresh;
            }
            Refresh();
        }

        private void OnDisable()
        {
            if (gameManager != null)
            {
                gameManager.StateChanged -= Refresh;
            }
        }

        private void Refresh()
        {
            if (gameManager == null)
            {
                return;
            }

            if (timerText != null)
            {
                int remaining = Mathf.CeilToInt(gameManager.RemainingTime);
                int minutes = remaining / 60;
                int seconds = remaining % 60;
                timerText.text = $"{minutes:00}:{seconds:00}  Keys {gameManager.CollectedKeyCount}/{gameManager.TotalKeyCount}";
            }

            if (statusText != null)
            {
                switch (gameManager.Status)
                {
                    case GameStatus.Cleared:
                        statusText.text = "CLEAR!";
                        break;
                    case GameStatus.Failed:
                        statusText.text = "FAILED";
                        break;
                    default:
                        statusText.text = gameManager.ExitOpen ? "EXIT OPEN!" : string.Empty;
                        break;
                }
            }
        }
    }
}
