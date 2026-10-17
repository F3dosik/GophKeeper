package io.github.f3dosik.shron.ui.secrets

import androidx.annotation.StringRes
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.AddCircle
import androidx.compose.material.icons.filled.Lock
import androidx.compose.material.icons.filled.MoreVert
import androidx.compose.material3.DropdownMenu
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.ListItem
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.material3.pulltorefresh.PullToRefreshBox
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.dp
import io.github.f3dosik.shron.R
import io.github.f3dosik.shron.mobile.Mobile
import java.text.DateFormat
import java.util.Date

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun SecretsScreen(
    vm: SecretsViewModel,
    login: String,
    onAdd: () -> Unit,
    onOpen: (SecretRef) -> Unit,
    onServerSettings: () -> Unit,
) {
    // Список загружается при каждом показе экрана (после разблокировки, после добавления
    // секрета) и забывается при уходе.
    DisposableEffect(Unit) {
        vm.load()
        onDispose(vm::forget)
    }
    var menu by remember { mutableStateOf(false) }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text(login) },
                actions = {
                    IconButton(onClick = onAdd) {
                        Icon(Icons.Filled.AddCircle, contentDescription = stringResource(R.string.action_add_secret))
                    }
                    IconButton(onClick = vm::lock) {
                        Icon(Icons.Filled.Lock, contentDescription = stringResource(R.string.action_lock))
                    }
                    Box {
                        IconButton(onClick = { menu = true }) {
                            Icon(Icons.Filled.MoreVert, contentDescription = stringResource(R.string.action_more))
                        }
                        DropdownMenu(expanded = menu, onDismissRequest = { menu = false }) {
                            DropdownMenuItem(
                                text = { Text(stringResource(R.string.server_settings)) },
                                onClick = { menu = false; onServerSettings() },
                            )
                            DropdownMenuItem(
                                text = { Text(stringResource(R.string.logout)) },
                                onClick = { menu = false; vm.logout() },
                            )
                        }
                    }
                },
            )
        },
    ) { padding ->
        PullToRefreshBox(
            isRefreshing = vm.busy,
            onRefresh = vm::load,
            modifier = Modifier
                .fillMaxSize()
                .padding(padding),
        ) {
            val items = vm.items
            val error = vm.error
            when {
                error != null && items.isNullOrEmpty() -> Placeholder(error.message, isError = true)
                items == null -> Unit
                items.isEmpty() -> Placeholder(stringResource(R.string.secrets_empty))
                else -> LazyColumn(Modifier.fillMaxSize()) {
                    if (error != null) {
                        item { Text(error.message, color = MaterialTheme.colorScheme.error, modifier = Modifier.padding(16.dp)) }
                    }
                    items(items, key = { it.type + "/" + it.name }) { item ->
                        SecretRow(item, onClick = { onOpen(SecretRef(item.name, item.type)) })
                        HorizontalDivider()
                    }
                }
            }
        }
    }
}

@Composable
private fun SecretRow(item: SecretItem, onClick: () -> Unit) {
    ListItem(
        modifier = Modifier.clickable(onClick = onClick),
        overlineContent = { Text(typeLabel(item.type)?.let { stringResource(it) } ?: item.type) },
        headlineContent = { Text(item.name) },
        supportingContent = item.metadata.takeIf { it.isNotBlank() }?.let { { Text(it, maxLines = 2) } },
        trailingContent = if (item.updatedAt > 0) {
            { Text(DateFormat.getDateInstance(DateFormat.SHORT).format(Date(item.updatedAt))) }
        } else null,
    )
}

/** Сообщение на месте списка; LazyColumn — чтобы работал жест обновления. */
@Composable
private fun Placeholder(text: String, isError: Boolean = false) {
    LazyColumn(Modifier.fillMaxSize(), horizontalAlignment = Alignment.CenterHorizontally) {
        item {
            Text(
                text,
                color = if (isError) MaterialTheme.colorScheme.error else MaterialTheme.colorScheme.onSurfaceVariant,
                modifier = Modifier.padding(32.dp),
            )
        }
    }
}

@StringRes
internal fun typeLabel(type: String): Int? = when (type) {
    Mobile.TypeCredentials -> R.string.type_credentials
    Mobile.TypeText -> R.string.type_text
    Mobile.TypeCard -> R.string.type_card
    Mobile.TypeBinary -> R.string.type_binary
    else -> null
}
