package io.github.f3dosik.shron.core

import org.junit.Assert.assertEquals
import org.junit.Assert.assertSame
import org.junit.Test

class ShronErrorTest {

    @Test
    fun `parses code and message from the Go error text`() {
        val e = ShronError.from(Exception("""{"code":"WRONG_PASSWORD","message":"Неверный мастер-пароль"}"""))
        assertEquals("WRONG_PASSWORD", e.code)
        assertEquals("Неверный мастер-пароль", e.message)
    }

    @Test
    fun `keeps text that is not JSON as an internal error`() {
        val e = ShronError.from(RuntimeException("go: something broke"))
        assertEquals("INTERNAL", e.code)
        assertEquals("go: something broke", e.message)
    }

    @Test
    fun `JSON without the expected fields is an internal error`() {
        assertEquals("INTERNAL", ShronError.from(Exception("""{"error":"x"}""")).code)
    }

    @Test
    fun `empty message gets a placeholder`() {
        val e = ShronError.from(RuntimeException())
        assertEquals("INTERNAL", e.code)
        assertEquals("Неизвестная ошибка", e.message)
    }

    @Test
    fun `already converted errors pass through`() {
        val original = ShronError("VALIDATION", "x")
        assertSame(original, ShronError.from(original))
    }
}
