package io.github.f3dosik.shron.core

import android.content.Context
import androidx.core.content.edit

/**
 * Настройки Android-приложения, которых нет в Go-ядре. Секретов здесь нет, поэтому
 * обычные SharedPreferences (резервное копирование данных приложения выключено).
 */
class AppPreferences(context: Context) {

    private val prefs = context.getSharedPreferences("shron", Context.MODE_PRIVATE)

    /** Через сколько секунд в фоне блокировать хранилище; 0 — сразу. */
    var backgroundLockSeconds: Int
        get() = prefs.getInt(KEY_BACKGROUND_LOCK, BackgroundLock.DEFAULT_DELAY_SECONDS)
            .takeIf { it in BackgroundLock.DELAY_OPTIONS_SECONDS } ?: BackgroundLock.DEFAULT_DELAY_SECONDS
        set(value) = prefs.edit { putInt(KEY_BACKGROUND_LOCK, value) }

    private companion object {
        const val KEY_BACKGROUND_LOCK = "background_lock_seconds"
    }
}
