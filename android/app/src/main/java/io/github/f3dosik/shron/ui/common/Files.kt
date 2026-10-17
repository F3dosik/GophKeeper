package io.github.f3dosik.shron.ui.common

import android.content.ContentResolver
import android.net.Uri
import android.provider.OpenableColumns
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import java.io.ByteArrayOutputStream

/** Файл, выбранный через системный диалог. */
class PickedFile(val name: String, val data: ByteArray)

/**
 * Читает файл, выбранный через Storage Access Framework. Возвращает null, если файл не
 * открылся; при размере больше [limit] — данные из [limit] + 1 байт, чтобы вызывающий
 * отличил «ровно limit» от «больше».
 */
suspend fun readPickedFile(resolver: ContentResolver, uri: Uri, limit: Int): PickedFile? =
    withContext(Dispatchers.IO) {
        val data = resolver.openInputStream(uri)?.use { input ->
            // InputStream.readNBytes есть только с API 33.
            val out = ByteArrayOutputStream()
            val chunk = ByteArray(64 * 1024)
            while (out.size() <= limit) {
                val n = input.read(chunk, 0, minOf(chunk.size, limit + 1 - out.size()))
                if (n < 0) break
                out.write(chunk, 0, n)
            }
            out.toByteArray()
        } ?: return@withContext null
        PickedFile(displayName(resolver, uri), data)
    }

private fun displayName(resolver: ContentResolver, uri: Uri): String =
    resolver.query(uri, arrayOf(OpenableColumns.DISPLAY_NAME), null, null, null)?.use { c ->
        if (c.moveToFirst()) c.getString(0) else null
    } ?: uri.lastPathSegment.orEmpty()
