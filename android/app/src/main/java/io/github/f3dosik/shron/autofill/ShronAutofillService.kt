package io.github.f3dosik.shron.autofill

import android.app.PendingIntent
import android.app.assist.AssistStructure
import android.content.Context
import android.content.Intent
import android.os.Build
import android.os.CancellationSignal
import android.service.autofill.AutofillService
import android.service.autofill.Dataset
import android.service.autofill.FillCallback
import android.service.autofill.FillRequest
import android.service.autofill.FillResponse
import android.service.autofill.SaveCallback
import android.service.autofill.SaveRequest
import android.view.View
import android.view.autofill.AutofillId
import android.view.autofill.AutofillValue
import android.widget.RemoteViews
import androidx.annotation.RequiresApi
import io.github.f3dosik.shron.R

/**
 * Служба автозаполнения. На форму входа она отвечает одним пунктом «Shron» без данных:
 * значения появляются, только когда пользователь выбрал логин в [AutofillPickerActivity]
 * (при необходимости разблокировав хранилище). Так служба ничего не раскрывает приложению,
 * которое выдаёт себя за чужой сайт, и одинаково работает с открытым и закрытым хранилищем.
 */
@RequiresApi(Build.VERSION_CODES.O)
class ShronAutofillService : AutofillService() {

    override fun onFillRequest(request: FillRequest, cancellationSignal: CancellationSignal, callback: FillCallback) {
        val structure = request.fillContexts.lastOrNull()?.structure
        val target = structure?.activityComponent?.packageName
        // Свои экраны не заполняем: там мастер-пароль и формы секретов.
        if (structure == null || target == null || target == packageName) {
            callback.onSuccess(null)
            return
        }

        val scan = Scan()
        for (i in 0 until structure.windowNodeCount) scan.visit(structure.getWindowNodeAt(i).rootViewNode)
        val form = parseLoginForm(scan.fields)
        if (form == null) {
            callback.onSuccess(null)
            return
        }

        val picker = Intent(this, AutofillPickerActivity::class.java)
            .putExtra(AutofillPickerActivity.EXTRA_USERNAME_ID, form.username)
            .putExtra(AutofillPickerActivity.EXTRA_PASSWORD_ID, form.password)
            .putExtra(AutofillPickerActivity.EXTRA_WEB_DOMAIN, scan.webDomain)
            .putExtra(AutofillPickerActivity.EXTRA_PACKAGE, target)
        // Неизменяемый PendingIntent: система не дописывает в него данные, всё нужное уже в
        // наших extras. Свой requestCode на каждый запрос — чтобы extras не перезаписывались.
        val sender = PendingIntent.getActivity(
            this,
            request.id,
            picker,
            PendingIntent.FLAG_CANCEL_CURRENT or PendingIntent.FLAG_IMMUTABLE,
        ).intentSender

        val dataset = datasetBuilder(this, getString(R.string.autofill_choose))
        listOfNotNull(form.username, form.password).forEach { dataset.setValueCompat(it, null) }
        dataset.setAuthentication(sender)
        callback.onSuccess(FillResponse.Builder().addDataset(dataset.build()).build())
    }

    override fun onSaveRequest(request: SaveRequest, callback: SaveCallback) {
        // Сохранение новых логинов пока не поддерживается: SaveInfo в ответах не задаётся,
        // поэтому система сюда не обращается.
        callback.onSuccess()
    }

    /** Поля формы и домен сайта (для браузеров) из структуры экрана. */
    private class Scan {
        val fields = mutableListOf<FormField<AutofillId>>()
        var webDomain: String? = null

        fun visit(node: AssistStructure.ViewNode) {
            if (webDomain == null && !node.webDomain.isNullOrBlank()) webDomain = node.webDomain
            val id = node.autofillId
            if (id != null && node.autofillType == View.AUTOFILL_TYPE_TEXT && node.visibility == View.VISIBLE) {
                fields += FormField(
                    id = id,
                    autofillHints = node.autofillHints?.toList().orEmpty(),
                    inputType = node.inputType,
                    idEntry = node.idEntry,
                    hint = node.hint,
                    html = node.htmlInfo?.attributes?.associate { it.first.lowercase() to it.second }.orEmpty(),
                )
            }
            for (i in 0 until node.childCount) visit(node.getChildAt(i))
        }
    }

    companion object {
        /** Строка в выпадающем списке автозаполнения. */
        fun presentation(context: Context, text: String) =
            RemoteViews(context.packageName, R.layout.autofill_item).apply {
                setTextViewText(R.id.autofill_text, text)
            }

        // Dataset.Builder(RemoteViews) и setValue(id, value) устарели в API 33 (Presentations и
        // Field), но работают на всех версиях; новый API — только с 33, а minSdk службы — 26.
        @Suppress("DEPRECATION")
        fun datasetBuilder(context: Context, text: String) = Dataset.Builder(presentation(context, text))

        @Suppress("DEPRECATION")
        fun Dataset.Builder.setValueCompat(id: AutofillId, value: AutofillValue?): Dataset.Builder =
            setValue(id, value)
    }
}
