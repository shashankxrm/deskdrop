package com.deskdrop.android

import android.content.Intent
import java.util.UUID

object ShareSubmit {
    fun payloadFromIntent(intent: Intent?): SharePayload? {
        val text = intent?.getStringExtra(Intent.EXTRA_TEXT).orEmpty()
        val subject = intent?.getStringExtra(Intent.EXTRA_SUBJECT)
        return ShareUrl.fromSend(text, subject)
    }

    fun newKey(): String = UUID.randomUUID().toString()
}
