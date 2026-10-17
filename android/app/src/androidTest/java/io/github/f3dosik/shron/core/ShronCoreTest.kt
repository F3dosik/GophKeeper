package io.github.f3dosik.shron.core

import androidx.test.ext.junit.runners.AndroidJUnit4
import androidx.test.platform.app.InstrumentationRegistry
import io.github.f3dosik.shron.mobile.Mobile
import kotlinx.coroutines.runBlocking
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Assert.fail
import org.junit.Before
import org.junit.Test
import org.junit.runner.RunWith
import java.io.File

/** ShronCore с настоящим Go-ядром из shron.aar; сервер не нужен. */
@RunWith(AndroidJUnit4::class)
class ShronCoreTest {

    private lateinit var dir: File

    @Before
    fun setUp() {
        val context = InstrumentationRegistry.getInstrumentation().targetContext
        dir = File(context.cacheDir, "core-test-${System.nanoTime()}").apply { mkdirs() }
    }

    @After
    fun tearDown() {
        dir.deleteRecursively()
    }

    private fun newCore() = ShronCore(dir, insecure = false)

    private fun expectError(code: String, block: suspend () -> Unit): ShronError {
        try {
            runBlocking { block() }
        } catch (e: ShronError) {
            assertEquals(e.message, code, e.code)
            return e
        }
        fail("expected ShronError $code")
        throw AssertionError()
    }

    @Test
    fun freshCoreIsNotConfigured() {
        val state = newCore().state.value
        assertFalse(state.configured)
        assertEquals("", state.login)
        assertFalse(state.unlocked)
    }

    @Test
    fun callsThatNeedServerFailWithoutSettings() {
        val core = newCore()
        expectError(Mobile.CodeNotConfigured) { core.listSecrets() }
    }

    @Test
    fun invalidAddressIsAValidationError() {
        val core = newCore()
        expectError(Mobile.CodeValidation) { core.call { it.saveSettings("http://", "", false, 5) } }
        assertFalse(core.state.value.configured)
    }

    @Test
    fun settingsArePersisted() {
        runBlocking { newCore().call { it.saveSettings("127.0.0.1", "", false, 7) } }

        val state = newCore().state.value
        assertTrue(state.configured)
        assertEquals("default port is added", "127.0.0.1:50051", state.serverAddress)
        assertEquals(7, state.autoLockMinutes)
        assertFalse(state.hasCACert)
    }

    @Test
    fun brokenCACertificateIsRejected() {
        val core = newCore()
        expectError(Mobile.CodeValidation) { core.call { it.saveSettings("127.0.0.1", "not a pem", false, 5) } }
    }

    @Test
    fun unreachableServerIsReported() {
        val core = newCore()
        runBlocking { core.call { it.saveSettings("127.0.0.1:1", "", false, 5) } }
        expectError(Mobile.CodeUnavailable) { core.call { it.signIn("alice", "password123") } }
        assertEquals("", core.state.value.login)
    }

    @Test
    fun lockingALockedVaultKeepsTheReason() {
        val core = newCore()
        core.lock(BackgroundLock.LOCK_REASON)
        assertNull("nothing was unlocked, so there is no reason to show", core.state.value.lockReason)
    }

    @Test
    fun passwordFollowsGeneratorSettings() = runBlocking {
        val core = newCore()

        val alnum = core.generate(GeneratorSettings(length = 33, symbols = false))
        assertEquals(33, alnum.password.length)
        assertTrue(alnum.password, alnum.password.all { it.isLetterOrDigit() })
        assertTrue(alnum.entropyBits > 0)

        val digits = core.generate(GeneratorSettings(length = 12, lower = false, upper = false, symbols = false))
        assertTrue(digits.password, digits.password.all { it.isDigit() })

        repeat(20) {
            val clear = core.generate(GeneratorSettings(length = 64, noAmbiguous = true)).password
            assertTrue(clear, clear.none { it in "0O1lI|" })
        }
    }

    @Test
    fun passphraseHasTheRequestedWordCount() = runBlocking {
        val result = newCore().generate(GeneratorSettings(passphrase = true, words = 5))
        assertEquals(result.password, 5, result.password.split("-").size)
    }

    @Test
    fun generatorWithoutCharacterSetsIsAValidationError() {
        val core = newCore()
        expectError(Mobile.CodeValidation) {
            core.generate(GeneratorSettings(lower = false, upper = false, digits = false, symbols = false))
        }
    }
}
