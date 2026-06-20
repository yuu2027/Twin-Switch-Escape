using UnityEngine;

namespace TwinSwitchEscape
{
    /// <summary>
    /// 鍵。プレイヤーが範囲内で Interact(E) を押すと取得済みになる。
    /// 取得状態は Collected で公開し、GameManager が収集数を集計する。
    /// </summary>
    public class KeyPickup : Interactable
    {
        [SerializeField] private string keyId = "key_1";
        [SerializeField] private SpriteRenderer spriteRenderer;

        /// <summary>このキーの識別子（spec の keyId に対応）。</summary>
        public string KeyId => keyId;

        /// <summary>取得済みかどうか。</summary>
        public bool Collected { get; private set; }

        private Collider2D ownCollider;

        private void Awake()
        {
            ownCollider = GetComponent<Collider2D>();
            if (spriteRenderer == null)
            {
                spriteRenderer = GetComponent<SpriteRenderer>();
            }
        }

        protected override void OnInteract(PlayerController player)
        {
            if (Collected)
            {
                return;
            }

            Collected = true;

            // 見た目を消し、再取得・再購読が起きないようにする。
            if (spriteRenderer != null)
            {
                spriteRenderer.enabled = false; // 非表示にする
            }
            if (ownCollider != null)
            {
                ownCollider.enabled = false; // 当たり判定を消す
            }
        }
    }
}
