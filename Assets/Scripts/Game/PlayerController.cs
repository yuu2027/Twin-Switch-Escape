using System;
using UnityEngine;
using UnityEngine.InputSystem;

namespace TwinSwitchEscape
{
    /// <summary>
    /// プレイヤーの移動とインタラクト入力を担当する。
    /// 入力は Unity Input System の PlayerInput(Send Messages) から受け取る。
    ///
    /// 表示（アニメーション）とは分離しており、本クラスは Animator を一切触らない。
    /// 移動方向や向きは MoveInput / Facing として公開し、
    /// アニメーションは後付けの PlayerAnimationBridge が参照する。
    /// </summary>
    [RequireComponent(typeof(Rigidbody2D))]
    public class PlayerController : MonoBehaviour
    {
        [Tooltip("移動速度（ユニット/秒）")]
        [SerializeField] private float moveSpeed = 4f;

        private Rigidbody2D body;
        private Vector2 moveInput;
        private Vector2 facing = Vector2.down;

        /// <summary>現在の移動入力ベクトル（正規化済み）。</summary>
        public Vector2 MoveInput => moveInput;

        /// <summary>最後に向いていた方向（停止中も保持）。アニメの向き決定に使う。</summary>
        public Vector2 Facing => facing;

        private void Awake()
        {
            body = GetComponent<Rigidbody2D>();
            // トップビューなので重力なし・回転固定を保証する。
            body.gravityScale = 0f;
            body.freezeRotation = true;
        }

        private void FixedUpdate()
        {
            body.linearVelocity = moveInput * moveSpeed;
        }

        // --- Input System (PlayerInput / Send Messages) からのコールバック ---
        private void OnMove(InputValue value)
        {
            moveInput = value.Get<Vector2>();
            if (moveInput.sqrMagnitude > 0.01f)
            {
                facing = moveInput.normalized;
            }
        }
    }
}
