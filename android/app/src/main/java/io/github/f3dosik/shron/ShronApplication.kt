package io.github.f3dosik.shron

import android.app.Application
import io.github.f3dosik.shron.core.SecureClipboard
import io.github.f3dosik.shron.core.ShronCore

class ShronApplication : Application() {

    /** Go-ядро создаётся при первом обращении: загрузка библиотеки и настроек — не в onCreate. */
    val core: ShronCore by lazy { ShronCore(filesDir, BuildConfig.SHRON_INSECURE) }

    val clipboard: SecureClipboard by lazy { SecureClipboard(this) }
}
