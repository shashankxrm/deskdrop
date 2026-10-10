package com.deskdrop.android

import android.content.ClipboardManager
import android.content.Intent
import android.os.Bundle
import android.os.Handler
import android.os.Looper
import android.view.View
import android.widget.Button
import android.widget.TextView
import android.widget.Toast
import androidx.appcompat.app.AppCompatActivity
import com.google.android.material.textfield.TextInputEditText
import com.google.android.material.textfield.TextInputLayout
import com.google.zxing.client.android.Intents
import com.journeyapps.barcodescanner.ScanContract
import com.journeyapps.barcodescanner.ScanOptions
import java.io.IOException
import java.util.concurrent.Executors

class PairActivity : AppCompatActivity() {
    private lateinit var apiUrl: TextInputEditText
    private lateinit var code: TextInputEditText
    private lateinit var status: TextView
    private lateinit var pair: Button
    private lateinit var unpair: Button
    private lateinit var store: SessionStore
    private val io = Executors.newSingleThreadExecutor()
    private val main = Handler(Looper.getMainLooper())

    private val qr = registerForActivityResult(ScanContract()) { result ->
        val text = result.contents ?: return@registerForActivityResult
        code.setText(PairingCode.normalize(text.removePrefix("deskdrop:")))
    }

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        if (intent?.action == Intent.ACTION_SEND) {
            startActivity(
                Intent(this, ShareActivity::class.java).apply {
                    action = Intent.ACTION_SEND
                    type = intent.type ?: "text/plain"
                    putExtras(intent)
                },
            )
            finish()
            return
        }
        setContentView(R.layout.activity_pair)
        store = SessionStore(this)
        apiUrl = findViewById(R.id.apiUrl)
        code = findViewById(R.id.code)
        status = findViewById(R.id.status)
        pair = findViewById(R.id.pair)
        unpair = findViewById(R.id.unpair)
        if (store.baseUrl.isNotEmpty()) {
            apiUrl.setText(store.baseUrl)
        }
        findViewById<Button>(R.id.paste).setOnClickListener { pasteCode() }
        findViewById<Button>(R.id.scan).setOnClickListener {
            qr.launch(
                ScanOptions()
                    .setDesiredBarcodeFormats(ScanOptions.QR_CODE)
                    .setPrompt(getString(R.string.code_label))
                    .addExtra(Intents.Scan.BARCODE_IMAGE_ENABLED, false),
            )
        }
        pair.setOnClickListener { startPair() }
        findViewById<Button>(R.id.sendLink).setOnClickListener { sendFromBox() }
        unpair.setOnClickListener {
            store.clear()
            renderPaired()
            status.text = "Forgot local token. Server device row is unchanged until revoke."
        }
        renderPaired()
    }

    private fun pasteCode() {
        val clip = getSystemService(ClipboardManager::class.java) ?: return
        val t = clip.primaryClip?.getItemAt(0)?.coerceToText(this)?.toString().orEmpty()
        if (t.isNotBlank()) {
            code.setText(PairingCode.normalize(t))
        }
    }

    private fun startPair() {
        val base = apiUrl.text?.toString().orEmpty().trim()
        val raw = PairingCode.normalize(code.text?.toString().orEmpty())
        if (base.isEmpty() || !base.startsWith("http://") && !base.startsWith("https://")) {
            status.setText(R.string.need_url)
            return
        }
        if (!PairingCode.looksComplete(raw)) {
            status.setText(R.string.need_code)
            return
        }
        store.baseUrl = base
        pair.isEnabled = false
        status.text = "Claiming…"
        io.execute {
            try {
                val res = RetryingClaim(DeskdropApi(), Backoff()).claim(base, raw, android.os.Build.MODEL)
                main.post {
                    pair.isEnabled = true
                    if (res.status == 201 && !res.token.isNullOrEmpty() && !res.deviceId.isNullOrEmpty()) {
                        store.saveClaim(base, res.token, res.deviceId)
                        renderPaired()
                    } else {
                        status.text = "Claim failed (${res.status}): ${res.error ?: "see pairing.md"}"
                    }
                }
            } catch (e: IOException) {
                main.post {
                    pair.isEnabled = true
                    status.text = "Network error: ${e.message}"
                }
            }
        }
    }

    private fun sendFromBox() {
        val raw = findViewById<TextInputEditText>(R.id.shareUrl).text?.toString().orEmpty()
        val payload = ShareUrl.fromSend(raw, null)
        if (payload == null) {
            status.setText(R.string.share_not_url)
            return
        }
        if (!store.isPaired) {
            status.setText(R.string.share_need_pair)
            return
        }
        status.setText(R.string.share_sending)
        val key = ShareSubmit.newKey()
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
                        status.text = getString(R.string.share_ok)
                        Toast.makeText(this, R.string.share_ok, Toast.LENGTH_SHORT).show()
                    } else {
                        status.text = "Share failed (${res.status}): ${res.error ?: ""}"
                    }
                }
            } catch (e: IOException) {
                main.post { status.text = "Network error: ${e.message}" }
            }
        }
    }

    private fun renderPaired() {
        val paired = store.isPaired
        unpair.visibility = if (paired) View.VISIBLE else View.GONE
        findViewById<TextInputLayout>(R.id.shareBox).visibility = if (paired) View.VISIBLE else View.GONE
        findViewById<Button>(R.id.sendLink).visibility = if (paired) View.VISIBLE else View.GONE
        pair.isEnabled = !paired
        if (paired) {
            status.text = getString(R.string.paired) + "\n" + store.deviceId
        }
    }
}
