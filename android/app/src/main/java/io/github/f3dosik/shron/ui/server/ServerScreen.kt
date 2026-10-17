package io.github.f3dosik.shron.ui.server

import androidx.activity.compose.BackHandler
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import io.github.f3dosik.shron.R
import io.github.f3dosik.shron.ui.common.ErrorMessage
import io.github.f3dosik.shron.ui.common.FormScreen
import io.github.f3dosik.shron.ui.common.InfoMessage
import io.github.f3dosik.shron.ui.common.PrimaryButton
import io.github.f3dosik.shron.ui.common.TextInput

/** Настройки сервера. [onClose] — null при первом запуске, когда возвращаться некуда. */
@Composable
fun ServerScreen(vm: ServerViewModel, onSaved: () -> Unit, onClose: (() -> Unit)?) {
    val context = LocalContext.current
    val pickCA = rememberLauncherForActivityResult(ActivityResultContracts.OpenDocument()) { uri ->
        if (uri != null) vm.loadCA(context.contentResolver, uri)
    }
    if (onClose != null) BackHandler(onBack = onClose)

    FormScreen(
        title = stringResource(R.string.server_title),
        subtitle = stringResource(R.string.server_subtitle),
    ) {
        TextInput(
            value = vm.address,
            onValueChange = { vm.address = it },
            label = stringResource(R.string.server_address),
            supportingText = stringResource(R.string.server_address_hint),
            keyboardType = KeyboardType.Uri,
            enabled = !vm.busy,
        )

        Column(verticalArrangement = Arrangement.spacedBy(4.dp)) {
            Text(stringResource(R.string.server_ca), style = MaterialTheme.typography.titleSmall)
            Text(
                when (val ca = vm.ca) {
                    CaSource.System -> stringResource(R.string.server_ca_system)
                    CaSource.Saved -> stringResource(R.string.server_ca_saved)
                    is CaSource.File -> stringResource(R.string.server_ca_file, ca.name)
                },
                style = MaterialTheme.typography.bodyMedium,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                OutlinedButton(onClick = { pickCA.launch(arrayOf("*/*")) }, enabled = !vm.busy) {
                    Text(stringResource(R.string.server_ca_pick))
                }
                if (vm.ca != CaSource.System) {
                    TextButton(onClick = vm::useSystemCA, enabled = !vm.busy) {
                        Text(stringResource(R.string.server_ca_use_system))
                    }
                }
            }
        }

        TextInput(
            value = vm.autoLockMinutes,
            onValueChange = { vm.autoLockMinutes = it.filter(Char::isDigit) },
            label = stringResource(R.string.server_auto_lock),
            supportingText = stringResource(R.string.server_auto_lock_hint),
            keyboardType = KeyboardType.Number,
            enabled = !vm.busy,
        )

        ErrorMessage(vm.error)
        InfoMessage(vm.checkResult)

        PrimaryButton(stringResource(R.string.save), busy = vm.busy) { vm.save(onSaved) }
        OutlinedButton(onClick = vm::check, enabled = !vm.busy, modifier = Modifier.fillMaxWidth()) {
            Text(stringResource(R.string.server_check))
        }
        if (onClose != null) {
            TextButton(onClick = onClose, modifier = Modifier.fillMaxWidth()) { Text(stringResource(R.string.cancel)) }
        }
    }
}
