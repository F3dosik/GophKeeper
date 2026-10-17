package io.github.f3dosik.shron.core

import io.github.f3dosik.shron.mobile.Mobile
import org.json.JSONException
import org.json.JSONObject

/**
 * Ошибка Go-ядра. gomobile передаёт в Kotlin только текст исключения, а ядро кладёт в него
 * JSON {"code": ..., "message": ...}: по [code] (константы Mobile.Code*) приложение решает,
 * что делать, [message] показывается пользователю.
 */
class ShronError(val code: String, override val message: String, cause: Throwable? = null) :
    Exception(message, cause) {

    companion object {
        fun from(e: Throwable): ShronError {
            if (e is ShronError) return e
            val text = e.message.orEmpty()
            return try {
                val json = JSONObject(text)
                ShronError(json.getString("code"), json.getString("message"), e)
            } catch (_: JSONException) {
                ShronError(Mobile.CodeInternal, text.ifBlank { "Неизвестная ошибка" }, e)
            }
        }
    }
}
