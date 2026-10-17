package io.github.f3dosik.shron.core

import org.junit.Assert.assertEquals
import org.junit.Before
import org.junit.Test

class BackgroundLockTest {

    /** Часы и таймер, которыми управляет тест. */
    private class FakeTime : BackgroundLock.Scheduler {
        var now = 0L
        private val pending = mutableMapOf<Runnable, Long>()

        override fun schedule(delayMs: Long, action: Runnable) {
            pending[action] = now + delayMs
        }

        override fun cancel(action: Runnable) {
            pending.remove(action)
        }

        /** Сдвигает время; [frozen] — процесс заморожен, и таймеры не срабатывают. */
        fun advance(ms: Long, frozen: Boolean = false) {
            now += ms
            if (frozen) return
            pending.filterValues { it <= now }.keys.forEach {
                pending.remove(it)
                it.run()
            }
        }
    }

    private lateinit var time: FakeTime
    private var delayMs = 30_000L
    private var locks = 0
    private lateinit var lock: BackgroundLock

    @Before
    fun setUp() {
        time = FakeTime()
        locks = 0
        lock = BackgroundLock(clock = { time.now }, scheduler = time, delayMs = { delayMs }, lock = { locks++ })
    }

    @Test
    fun `zero delay locks as soon as the app goes to background`() {
        delayMs = 0
        lock.onBackground()
        assertEquals(1, locks)
    }

    @Test
    fun `timer locks after the delay`() {
        lock.onBackground()
        time.advance(29_999)
        assertEquals(0, locks)
        time.advance(1)
        assertEquals(1, locks)
    }

    @Test
    fun `returning before the delay keeps the vault unlocked`() {
        lock.onBackground()
        time.advance(10_000)
        lock.onForeground()
        time.advance(60_000)
        assertEquals(0, locks)
    }

    @Test
    fun `frozen process locks on return when the delay has passed`() {
        lock.onBackground()
        time.advance(5 * 60_000, frozen = true)
        assertEquals("the timer could not fire", 0, locks)
        lock.onForeground()
        assertEquals(1, locks)
    }

    @Test
    fun `own system dialog extends the delay`() {
        delayMs = 0
        lock.expectExternalActivity()
        lock.onBackground()
        time.advance(BackgroundLock.EXTERNAL_ACTIVITY_MS - 1)
        lock.onForeground()
        assertEquals(0, locks)
    }

    @Test
    fun `own system dialog left open too long still locks`() {
        lock.expectExternalActivity()
        lock.onBackground()
        time.advance(BackgroundLock.EXTERNAL_ACTIVITY_MS)
        assertEquals(1, locks)
    }

    @Test
    fun `system dialog allowance ends after returning`() {
        delayMs = 0
        lock.expectExternalActivity()
        lock.onBackground()
        lock.onForeground()

        lock.onBackground()
        assertEquals("the next background trip uses the normal delay", 1, locks)
    }

    @Test
    fun `delay is read when the app goes to background`() {
        lock.onBackground()
        lock.onForeground()
        delayMs = 0
        lock.onBackground()
        assertEquals(1, locks)
    }
}
