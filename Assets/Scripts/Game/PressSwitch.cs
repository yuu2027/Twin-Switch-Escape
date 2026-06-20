using UnityEngine;

namespace TwinSwitchEscape
{
    /// <summary>
    /// 床スイッチ。プレイヤーが乗っている間 ON。
    /// 離れても holdSeconds の間は ON を保持する（1人プレイで「同時押し」を簡易再現するため）。
    ///
    /// spec §9.5 の「範囲内にプレイヤーがいれば pressedBy」を踏襲しつつ、
    /// Phase 1 ローカル版では保持時間を設けて 1 人でも両スイッチを成立させられるようにしている。
    /// </summary>
    [RequireComponent(typeof(Collider2D))]
    public class PressSwitch : MonoBehaviour
    {
        [SerializeField] private string switchId = "switch_a";
        [Tooltip("プレイヤーが離れてから ON を保持する秒数（同時押し簡易再現の猶予）")]
        [SerializeField] private float holdSeconds = 7f;

        [Header("表示（任意）")]
        [SerializeField] private SpriteRenderer spriteRenderer;
        [SerializeField] private Sprite pressedSprite;
        [SerializeField] private Sprite releasedSprite;

        /// <summary>このスイッチの識別子（spec の switchId に対応）。</summary>
        public string SwitchId => switchId;

        /// <summary>現在 ON かどうか（保持中も含む）。</summary>
        public bool IsPressed { get; private set; }

        private int occupants;
        private float releaseTimer;

        private void Awake()
        {
            if (spriteRenderer == null)
            {
                spriteRenderer = GetComponent<SpriteRenderer>();
            }
            ApplySprite();
        }

        private void Update()
        {
            bool occupied = occupants > 0;

            if (occupied)
            {
                SetPressed(true);
                releaseTimer = holdSeconds;
            }
            else if (IsPressed)
            {
                releaseTimer -= Time.deltaTime;
                if (releaseTimer <= 0f)
                {
                    SetPressed(false);
                }
            }
        }

        private void OnTriggerEnter2D(Collider2D other)
        {
            if (other.GetComponentInParent<PlayerController>() != null)
            {
                occupants++;
            }
        }

        private void OnTriggerExit2D(Collider2D other)
        {
            if (other.GetComponentInParent<PlayerController>() != null)
            {
                occupants = Mathf.Max(0, occupants - 1);
            }
        }

        private void SetPressed(bool pressed)
        {
            if (IsPressed == pressed)
            {
                return;
            }
            IsPressed = pressed;
            ApplySprite();
        }

        private void ApplySprite()
        {
            if (spriteRenderer == null)
            {
                return;
            }
            Sprite next = IsPressed ? pressedSprite : releasedSprite;
            if (next != null)
            {
                spriteRenderer.sprite = next;
            }
        }
    }
}
