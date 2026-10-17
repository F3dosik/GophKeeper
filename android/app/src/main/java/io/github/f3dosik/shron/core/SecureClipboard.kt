package io.github.f3dosik.shron.core

import android.content.ClipData
import android.content.ClipboardManager
import android.content.Context
import android.os.Build
import android.os.Handler
import android.os.Looper
import android.os.PersistableBundle

/**
 * Копирование секретов в буфер обмена, как на десктопе: значение помечается
 * чувствительным (Android 13+ не показывает его в превью и подсказках клавиатуры)
 * и стирается через [CLEAR_AFTER_MS].
 */
class SecureClipboard(context: Context) {

    private val clipboard = context.getSystemService(ClipboardManager::class.java)
    private val handler = Handler(Looper.getMainLooper())
    private val clear = Runnable(::clearIfOurs)

    fun copy(value: String) {
        val clip = ClipData.newPlainText(LABEL, value)
        clip.description.extras = PersistableBundle().apply { putBoolean(EXTRA_IS_SENSITIVE, true) }
        clipboard.setPrimaryClip(clip)
        handler.removeCallbacks(clear)
        handler.postDelayed(clear, CLEAR_AFTER_MS)
    }

    /**
     * Стирает буфер, если там не скопированное позже из другого приложения. В фоне
     * Android 10+ не даёт прочитать буфер (описание — null), тогда буфер стирается.
     */
    private fun clearIfOurs() {
        val label = clipboard.primaryClipDescription?.label
        if (label != null && label != LABEL) return
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.P) {
            clipboard.clearPrimaryClip()
        } else {
            clipboard.setPrimaryClip(ClipData.newPlainText("", ""))
        }
    }

    companion object {
        const val CLEAR_AFTER_MS = 30_000L
        private const val LABEL = "Shron"

        // ClipDescription.EXTRA_IS_SENSITIVE появилась в API 33; на старых версиях флаг игнорируется.
        private const val EXTRA_IS_SENSITIVE = "android.content.extra.IS_SENSITIVE"
    }
}
