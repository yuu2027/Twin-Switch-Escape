using UnityEngine;

namespace TwinSwitchEscape
{
    /// <summary>
    /// プレイヤーが範囲内に入っている間に Interact(E) を押すと反応する対象の基底クラス。
    /// トリガ Collider2D を必要とし、範囲内プレイヤーの Interacted イベントを購読する。
    /// </summary>
    [RequireComponent(typeof(Collider2D))]
    public abstract class Interactable : MonoBehaviour
    {
        private PlayerController playerInRange;

        protected virtual void OnTriggerEnter2D(Collider2D other)
        {
            var player = other.GetComponentInParent<PlayerController>();
            if (player == null || playerInRange == player)
            {
                return;
            }

            OnInteract(player);
        }

        /// <summary>範囲内プレイヤーが Interact を押したときに呼ばれる。</summary>
        protected abstract void OnInteract(PlayerController player);
    }
}
