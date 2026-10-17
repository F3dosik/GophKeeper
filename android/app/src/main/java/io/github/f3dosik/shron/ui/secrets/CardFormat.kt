package io.github.f3dosik.shron.ui.secrets

/** Скрытое значение секрета. */
internal const val MASK = "••••••••••"

/** Номер карты группами по 4 цифры. */
internal fun formatCardNumber(number: String) = number.chunked(4).joinToString(" ")

/** Номер карты с последними 4 цифрами, как на десктопе; короткий номер скрывается целиком. */
internal fun maskCardNumber(number: String) =
    if (number.length > 4) "•••• " + number.takeLast(4) else MASK
