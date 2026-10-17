package io.github.f3dosik.shron.ui.common

import androidx.compose.runtime.Composable
import androidx.compose.runtime.remember
import androidx.compose.ui.platform.LocalContext
import io.github.f3dosik.shron.ShronApplication

/**
 * Вызывается перед запуском системного экрана (выбор или сохранение файла): приложение
 * уходит в фон, но хранилище не должно заблокироваться, пока пользователь выбирает файл.
 */
@Composable
fun rememberExternalActivity(): () -> Unit {
    val app = LocalContext.current.applicationContext as ShronApplication
    return remember(app) { app.backgroundLock::expectExternalActivity }
}
