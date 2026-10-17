package io.github.f3dosik.shron.ui.secrets

import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import io.github.f3dosik.shron.ShronApplication
import io.github.f3dosik.shron.ui.common.CoreViewModel

/** Строка списка секретов: копия SecretSummary, чтобы не держать ссылки на объекты Go. */
data class SecretItem(
    val name: String,
    val type: String,
    val metadata: String,
    val updatedAt: Long,
)

/**
 * Поиск как на десктопе: [query] — подстрока названия или комментария без учёта регистра,
 * [type] — тип секрета (null — все).
 */
fun filterSecrets(items: List<SecretItem>, query: String, type: String?): List<SecretItem> {
    val q = query.trim()
    return items.filter { item ->
        (type == null || item.type == type) &&
            (q.isEmpty() || item.name.contains(q, ignoreCase = true) || item.metadata.contains(q, ignoreCase = true))
    }
}

class SecretsViewModel(app: ShronApplication) : CoreViewModel(app) {

    var items by mutableStateOf<List<SecretItem>?>(null)
        private set
    var query by mutableStateOf("")
    var typeFilter by mutableStateOf<String?>(null)

    fun load() = run {
        items = core.listSecrets()
            .map { SecretItem(it.name, it.type, it.metadata, it.updatedAt) }
            .sortedWith(compareBy(String.CASE_INSENSITIVE_ORDER) { it.name })
    }

    /** Забывает список, когда хранилище блокируется. */
    fun forget() {
        items = null
        query = ""
        typeFilter = null
        dismissError()
    }

    fun lock() = core.lock()

    fun logout() = run { core.call { it.logout(false) } }
}
