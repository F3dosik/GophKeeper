package io.github.f3dosik.shron.ui.secrets

import org.junit.Assert.assertEquals
import org.junit.Test

class CardFormatTest {

    @Test
    fun `formats the number in groups of four`() {
        assertEquals("4111 1111 1111 1111", formatCardNumber("4111111111111111"))
        assertEquals("3782 8224 6310 005", formatCardNumber("378282246310005"))
        assertEquals("", formatCardNumber(""))
    }

    @Test
    fun `masks all but the last four digits`() {
        assertEquals("•••• 1111", maskCardNumber("4111111111111111"))
    }

    @Test
    fun `short numbers are hidden entirely`() {
        assertEquals(MASK, maskCardNumber("1234"))
        assertEquals(MASK, maskCardNumber(""))
    }
}
