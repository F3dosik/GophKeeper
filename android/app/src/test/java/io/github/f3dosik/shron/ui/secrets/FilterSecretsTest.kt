package io.github.f3dosik.shron.ui.secrets

import org.junit.Assert.assertEquals
import org.junit.Test

class FilterSecretsTest {

    private val mail = SecretItem("Почта", "credentials", "личный ящик", 0)
    private val bank = SecretItem("Банк", "card", "зарплатная", 0)
    private val note = SecretItem("Wi-Fi дома", "text", "", 0)
    private val all = listOf(mail, bank, note)

    private fun names(query: String, type: String? = null) = filterSecrets(all, query, type).map { it.name }

    @Test
    fun `empty query and no type keep everything`() {
        assertEquals(listOf("Почта", "Банк", "Wi-Fi дома"), names(""))
    }

    @Test
    fun `query matches name ignoring case`() {
        assertEquals(listOf("Почта"), names("почт"))
        assertEquals(listOf("Wi-Fi дома"), names("WI-FI"))
    }

    @Test
    fun `query matches the comment`() {
        assertEquals(listOf("Банк"), names("Зарплат"))
    }

    @Test
    fun `surrounding spaces are ignored`() {
        assertEquals(listOf("Банк"), names("  банк "))
    }

    @Test
    fun `type filter combines with the query`() {
        assertEquals(listOf("Банк"), names("", "card"))
        assertEquals(emptyList<String>(), names("почта", "card"))
    }
}
