package io.github.f3dosik.shron.core

/**
 * Блокировка хранилища, когда приложение ушло в фон: свернули, переключились на другое
 * приложение, выключили экран.
 *
 * Хранилище блокируется через [delayMs] после ухода в фон. Таймер в фоне может не сработать:
 * Android замораживает процессы свёрнутых приложений. Поэтому время ухода запоминается, и при
 * возвращении хранилище блокируется сразу, если задержка уже прошла, — до того, как
 * пользователь увидит секреты.
 *
 * Системные диалоги, которые открывает само приложение (выбор файла, сохранение), тоже
 * уводят его в фон; перед ними вызывается [expectExternalActivity], и задержка на это время
 * не меньше [EXTERNAL_ACTIVITY_MS].
 *
 * Класс не зависит от Android: часы и таймер передаются снаружи, чтобы проверять его в тестах.
 */
class BackgroundLock(
    private val clock: () -> Long,
    private val scheduler: Scheduler,
    private val delayMs: () -> Long,
    private val lock: () -> Unit,
) {
    /** Отложенный запуск в главном потоке (на Android — Handler). */
    interface Scheduler {
        fun schedule(delayMs: Long, action: Runnable)
        fun cancel(action: Runnable)
    }

    private val timer = Runnable { lock() }
    private var backgroundSince: Long? = null
    private var externalActivity = false

    /** Приложение само открывает системный экран и скоро вернётся. */
    fun expectExternalActivity() {
        externalActivity = true
    }

    fun onBackground() {
        backgroundSince = clock()
        val delay = effectiveDelay()
        if (delay <= 0) lock() else scheduler.schedule(delay, timer)
    }

    fun onForeground() {
        scheduler.cancel(timer)
        val since = backgroundSince
        if (since != null && clock() - since >= effectiveDelay()) lock()
        backgroundSince = null
        externalActivity = false
    }

    private fun effectiveDelay(): Long =
        if (externalActivity) maxOf(delayMs(), EXTERNAL_ACTIVITY_MS) else delayMs()

    companion object {
        /** Сколько ждать, пока открыт системный диалог приложения. */
        const val EXTERNAL_ACTIVITY_MS = 5 * 60_000L

        /** Варианты задержки в настройках, секунды; 0 — сразу. */
        val DELAY_OPTIONS_SECONDS = listOf(0, 30, 60, 300)
        const val DEFAULT_DELAY_SECONDS = 30

        /** Причина блокировки для CoreState.lockReason (Go-ядро о ней не знает). */
        const val LOCK_REASON = "background"
    }
}
