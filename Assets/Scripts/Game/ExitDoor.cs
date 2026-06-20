using UnityEngine;

namespace TwinSwitchEscape
{
    /// <summary>
    /// 出口の扉。GameManager から SetOpen(true) で開放される。
    /// 開いている状態でプレイヤーが範囲内にいるか（PlayerInside）を公開し、
    /// クリア判定は GameManager 側で行う。
    /// </summary>
    [RequireComponent(typeof(Collider2D))]
    public class ExitDoor : MonoBehaviour
    {
        [SerializeField] private SpriteRenderer spriteRenderer;
        [SerializeField] private Sprite openSprite;
        [SerializeField] private Sprite closedSprite;

        private Collider2D owncollider;

        /// <summary>出口が開いているか。</summary>
        public bool IsOpen { get; private set; }

        /// <summary>プレイヤーが出口範囲内にいるか。</summary>
        public bool PlayerInside { get; private set; }

        private void Awake()
        {
            if (spriteRenderer == null)
            {
                spriteRenderer = GetComponent<SpriteRenderer>();
            }
            owncollider = GetComponent<Collider2D>();
            ApplySprite();
        }

        public void SetOpen(bool open)
        {
            if (IsOpen == open) return;

            IsOpen = open;
            ApplySprite();
        }

        private void OnTriggerEnter2D(Collider2D other)
        {
            if (other.GetComponentInParent<PlayerController>() != null)
            {
                PlayerInside = true;
            }
        }

        private void OnTriggerExit2D(Collider2D other)
        {
            if (other.GetComponentInParent<PlayerController>() != null)
            {
                PlayerInside = false;
            }
        }

        private void ApplySprite()
        {
            if (spriteRenderer == null)
            {
                return;
            }
            Sprite next = IsOpen ? openSprite : closedSprite;
            if (next != null)
            {
                spriteRenderer.sprite = next;
                owncollider.enabled = !IsOpen;
            }
        }
    }
}
