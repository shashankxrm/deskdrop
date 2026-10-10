package com.deskdrop.android

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Test
import org.junit.runner.RunWith
import org.junit.runners.Parameterized

@RunWith(Parameterized::class)
class ShareUrlParseTest(
    private val raw: String,
    private val ok: Boolean,
) {
    @Test
    fun parse() {
        val got = ShareUrl.parse(raw)
        if (ok) {
            if (got == null) {
                throw AssertionError("expected ok for $raw")
            }
        } else {
            assertNull(raw, got)
        }
    }

    companion object {
        @JvmStatic
        @Parameterized.Parameters(name = "{0} ok={1}")
        fun data(): List<Array<Any>> = listOf(
            arrayOf("https://example.com/a", true),
            arrayOf("http://example.com/x", true),
            arrayOf("https://youtu.be/dQw4w9wgXcQ", true),
            arrayOf("ftp://example.com", false),
            arrayOf("javascript:alert(1)", false),
            arrayOf("https://", false),
            arrayOf("https://example.com/a b", false),
            arrayOf("", false),
            arrayOf("not a url", false),
        )
    }
}

class ShareUrlExtractTest {
    @Test
    fun youtubeShareSheet() {
        val p = ShareUrl.fromSend(
            "Rick Astley - Never Gonna Give You Up https://youtu.be/dQw4w9wgXcQ",
            "Rick Astley - Never Gonna Give You Up",
        )
        assertEquals("https://youtu.be/dQw4w9wgXcQ", p?.url)
        assertEquals("Rick Astley - Never Gonna Give You Up", p?.title)
    }

    @Test
    fun urlOnlyText() {
        val p = ShareUrl.fromSend("https://example.com/from-phone", null)
        assertEquals("https://example.com/from-phone", p?.url)
        assertEquals("", p?.title)
    }

    @Test
    fun rejectsNonUrl() {
        assertNull(ShareUrl.fromSend("hello world", "notes"))
    }
}

class RetryingShareTest {
    @Test
    fun sameKeyOnRetryAfter429() {
        var n = 0
        var keys = mutableListOf<String>()
        val api = object : DeskdropApi() {
            override fun share(
                baseUrl: String,
                token: String,
                url: String,
                title: String,
                idempotencyKey: String,
            ): ShareResult {
                n++
                keys.add(idempotencyKey)
                if (n == 1) return ShareResult(429, null, "rate limited")
                return ShareResult(201, "link-id", null)
            }
        }
        val res = RetryingShare(api, Backoff(initialMs = 1, random = { 0.0 }), sleep = {}).share(
            "http://127.0.0.1:8080",
            "tok",
            "https://example.com/a",
            "",
            "same-tap-key",
        )
        assertEquals(201, res.status)
        assertEquals(listOf("same-tap-key", "same-tap-key"), keys)
    }
}
