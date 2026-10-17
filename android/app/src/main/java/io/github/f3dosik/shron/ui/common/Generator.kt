package io.github.f3dosik.shron.ui.common

import androidx.annotation.StringRes
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ExperimentalLayoutApi
import androidx.compose.foundation.layout.FlowRow
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.FilterChip
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.SegmentedButton
import androidx.compose.material3.SegmentedButtonDefaults
import androidx.compose.material3.SingleChoiceSegmentedButtonRow
import androidx.compose.material3.Slider
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.dp
import io.github.f3dosik.shron.R
import io.github.f3dosik.shron.core.GeneratorSettings
import kotlin.math.roundToInt

/**
 * Настройки генератора: режим, длина или число слов, наборы символов.
 * [onChange] получает новые настройки и признак, что пароль пора пересоздать: переключатели
 * пересоздают его сразу, ползунок — когда его отпустили ([onCommit]).
 */
@OptIn(ExperimentalMaterial3Api::class, ExperimentalLayoutApi::class)
@Composable
fun GeneratorPanel(
    settings: GeneratorSettings,
    onChange: (settings: GeneratorSettings, regenerate: Boolean) -> Unit,
    onCommit: () -> Unit,
    enabled: Boolean = true,
) {
    Column(Modifier.fillMaxWidth(), verticalArrangement = Arrangement.spacedBy(4.dp)) {
        SingleChoiceSegmentedButtonRow(Modifier.fillMaxWidth()) {
            listOf(false to R.string.generator_chars, true to R.string.generator_passphrase)
                .forEachIndexed { index, (passphrase, label) ->
                    SegmentedButton(
                        selected = settings.passphrase == passphrase,
                        onClick = { onChange(settings.copy(passphrase = passphrase), true) },
                        shape = SegmentedButtonDefaults.itemShape(index, 2),
                        enabled = enabled,
                    ) { Text(stringResource(label)) }
                }
        }

        if (settings.passphrase) {
            CountSlider(
                label = stringResource(R.string.generator_words, settings.words),
                value = settings.words,
                range = GeneratorSettings.WORDS_RANGE,
                onValueChange = { onChange(settings.copy(words = it), false) },
                onCommit = onCommit,
                enabled = enabled,
            )
        } else {
            CountSlider(
                label = stringResource(R.string.generator_length, settings.length),
                value = settings.length,
                range = GeneratorSettings.LENGTH_RANGE,
                onValueChange = { onChange(settings.copy(length = it), false) },
                onCommit = onCommit,
                enabled = enabled,
            )
            val sets = listOf(settings.lower, settings.upper, settings.digits, settings.symbols)
            // Последний включённый набор выключить нельзя: генерировать было бы не из чего.
            fun canToggle(on: Boolean) = enabled && (!on || sets.count { it } > 1)
            FlowRow(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                Chip(R.string.generator_lower, settings.lower, canToggle(settings.lower)) {
                    onChange(settings.copy(lower = it), true)
                }
                Chip(R.string.generator_upper, settings.upper, canToggle(settings.upper)) {
                    onChange(settings.copy(upper = it), true)
                }
                Chip(R.string.generator_digits, settings.digits, canToggle(settings.digits)) {
                    onChange(settings.copy(digits = it), true)
                }
                Chip(R.string.generator_symbols, settings.symbols, canToggle(settings.symbols)) {
                    onChange(settings.copy(symbols = it), true)
                }
                Chip(R.string.generator_no_ambiguous, settings.noAmbiguous, enabled) {
                    onChange(settings.copy(noAmbiguous = it), true)
                }
            }
        }
    }
}

@Composable
private fun CountSlider(
    label: String,
    value: Int,
    range: IntRange,
    onValueChange: (Int) -> Unit,
    onCommit: () -> Unit,
    enabled: Boolean,
) {
    Text(label, style = MaterialTheme.typography.bodyMedium)
    Slider(
        value = value.toFloat(),
        onValueChange = { onValueChange(it.roundToInt()) },
        onValueChangeFinished = onCommit,
        valueRange = range.first.toFloat()..range.last.toFloat(),
        steps = range.last - range.first - 1,
        enabled = enabled,
    )
}

@Composable
private fun Chip(@StringRes label: Int, selected: Boolean, enabled: Boolean, onChange: (Boolean) -> Unit) {
    FilterChip(
        selected = selected,
        onClick = { onChange(!selected) },
        label = { Text(stringResource(label)) },
        enabled = enabled,
    )
}
