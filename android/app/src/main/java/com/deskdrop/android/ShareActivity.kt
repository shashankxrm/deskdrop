package com.deskdrop.android

import android.content.Intent
import android.os.Bundle
import android.os.Handler
import android.os.Looper
import android.widget.TextView
import android.widget.Toast
import androidx.appcompat.app.AppCompatActivity
import java.io.IOException
import java.util.UUID
import java.util.concurrent.Executors

class ShareActivity : AppCompatActivity() {
    private val io = Executors.newSingleThreadExecutor()
    private val main = Handler(Looper.getMainLooper())
    private lateinit var key: String

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContentView(R.layout.activity_share)
        val status = findViewById<TextView>(R.id.shareStatus)
        key = savedInstanceState?.getString(STATE_KEY) ?: UUID.randomUUID().toString()

        val store = SessionStore(this)
        if (!store.isPaired) {
            status.setText(R.string.share_need_pair)
            startActivity(Intent(this, PairActivity::class.java))
            finish()
            return
        }

        val payload = ShareSubmit.payloadFromIntent(intent)
        if (payload == null) {
            status.setText(R.string.share_not_url)
            Toast.makeText(this, R.string.share_not_url, Toast.LENGTH_LONG).show()
            finish()
            return
        }

        status.setText(R.string.share_sending)
        io.execute {
            try {
                val res = RetryingShare(DeskdropApi(), Backoff()).share(
                    store.baseUrl,
                    store.token,
                    payload.url,
                    payload.title,
                    key,
                )
                main.post {
                    if (res.ok) {
                        Toast.makeText(this, R.string.share_ok, Toast.LENGTH_SHORT).show()
                    } else {
                        Toast.makeText(this, "Share failed (${res.status}): ${res.error ?: ""}", Toast.LENGTH_LONG).show()
                    }
                    finish()
                }
            } catch (e: IOException) {
                main.post {
                    Toast.makeText(this, "Network error: ${e.message}", Toast.LENGTH_LONG).show()
                    finish()
                }
            }
        }
    }

    override fun onSaveInstanceState(outState: Bundle) {
        super.onSaveInstanceState(outState)
        outState.putString(STATE_KEY, key)
    }

    companion object {
        private const val STATE_KEY = "idempotency_key"
    }
}
