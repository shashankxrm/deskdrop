package com.deskdrop.android

import java.io.IOException
import java.net.HttpURLConnection
import java.net.URI
import java.nio.charset.StandardCharsets

data class ClaimResult(
    val status: Int,
    val token: String?,
    val deviceId: String?,
    val error: String?,
)

open class DeskdropApi(
    private val connectMs: Int = 8_000,
    private val readMs: Int = 15_000,
) {
    open fun claim(baseUrl: String, code: String, name: String = ""): ClaimResult {
        val root = baseUrl.trim().trimEnd('/')
        val uri = URI.create("$root/v1/pairings/claim")
        val body = """{"code":${jsonQuote(code)},"type":"phone","name":${jsonQuote(name)}}"""
        val conn = (uri.toURL().openConnection() as HttpURLConnection).apply {
            requestMethod = "POST"
            connectTimeout = connectMs
            readTimeout = readMs
            doOutput = true
            setRequestProperty("Content-Type", "application/json")
            setRequestProperty("Accept", "application/json")
        }
        try {
            conn.outputStream.use { it.write(body.toByteArray(StandardCharsets.UTF_8)) }
            val status = conn.responseCode
            val raw = (if (status in 200..299) conn.inputStream else conn.errorStream)
                ?.bufferedReader(StandardCharsets.UTF_8)?.readText().orEmpty()
            return ClaimResult(
                status = status,
                token = JsonStrings.string(raw, "token"),
                deviceId = JsonStrings.string(raw, "device_id"),
                error = JsonStrings.string(raw, "error"),
            )
        } finally {
            conn.disconnect()
        }
    }

    companion object {
        fun jsonQuote(s: String): String {
            val b = StringBuilder("\"")
            for (c in s) {
                when (c) {
                    '\\' -> b.append("\\\\")
                    '"' -> b.append("\\\"")
                    '\n' -> b.append("\\n")
                    '\r' -> b.append("\\r")
                    else -> b.append(c)
                }
            }
            b.append('"')
            return b.toString()
        }
    }
}

class RetryingClaim(
    private val api: DeskdropApi,
    private val backoff: Backoff,
    private val maxAttempts: Int = 5,
    private val sleep: (Long) -> Unit = { ms -> Thread.sleep(ms) },
) {
    fun claim(baseUrl: String, code: String, name: String = ""): ClaimResult {
        var last: ClaimResult? = null
        var lastIo: IOException? = null
        repeat(maxAttempts) { attempt ->
            try {
                val res = api.claim(baseUrl, code, name)
                last = res
                lastIo = null
                if (!Backoff.shouldRetry(res.status, false)) {
                    return res
                }
            } catch (e: IOException) {
                lastIo = e
                if (!Backoff.shouldRetry(null, true)) {
                    throw e
                }
            }
            if (attempt < maxAttempts - 1) {
                sleep(backoff.delayMs(attempt))
            }
        }
        lastIo?.let { throw it }
        return last ?: ClaimResult(0, null, null, "no response")
    }
}
