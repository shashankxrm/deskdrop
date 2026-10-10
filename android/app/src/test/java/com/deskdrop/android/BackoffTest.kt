package com.deskdrop.android

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test
import org.junit.runner.RunWith
import org.junit.runners.Parameterized

class BackoffTest {
    @Test
    fun delayIsZeroWhenRandomIsZero() {
        val b = Backoff(initialMs = 500, maxMs = 60_000, random = { 0.0 })
        assertEquals(0, b.delayMs(0))
        assertEquals(0, b.delayMs(8))
    }

    @Test
    fun delayHitsBaseOnAttemptZeroWhenRandomIsOne() {
        val b = Backoff(initialMs = 500, maxMs = 60_000, random = { 1.0 })
        assertEquals(500, b.delayMs(0))
    }

    @Test
    fun delayCapsAtMax() {
        val b = Backoff(initialMs = 500, maxMs = 8_000, random = { 1.0 })
        assertEquals(8_000, b.delayMs(10))
    }

    @Test
    fun delayDoublesUntilCap() {
        val b = Backoff(initialMs = 500, maxMs = 60_000, random = { 1.0 })
        assertEquals(500, b.delayMs(0))
        assertEquals(1_000, b.delayMs(1))
        assertEquals(2_000, b.delayMs(2))
        assertEquals(4_000, b.delayMs(3))
    }
}

@RunWith(Parameterized::class)
class BackoffRetryTest(
    private val status: Int?,
    private val ioError: Boolean,
    private val retry: Boolean,
) {
    @Test
    fun shouldRetry() {
        assertEquals(retry, Backoff.shouldRetry(status, ioError))
    }

    companion object {
        @JvmStatic
        @Parameterized.Parameters(name = "status={0} io={1} retry={2}")
        fun data(): List<Array<Any?>> = listOf(
            arrayOf(201, false, false),
            arrayOf(400, false, false),
            arrayOf(401, false, false),
            arrayOf(404, false, false),
            arrayOf(409, false, false),
            arrayOf(410, false, false),
            arrayOf(429, false, true),
            arrayOf(500, false, true),
            arrayOf(502, false, true),
            arrayOf(503, false, true),
            arrayOf(null, true, true),
            arrayOf(null, false, false),
        )
    }
}

class PairingCodeTest {
    @Test
    fun normalizeInsertsHyphen() {
        assertEquals("ABCD-EFGH", PairingCode.normalize("abcd efgh"))
        assertEquals("ABCD-EFGH", PairingCode.normalize("ABCD-EFGH"))
        assertTrue(PairingCode.looksComplete("abcd-efgh"))
        assertFalse(PairingCode.looksComplete("nope"))
    }
}

class JsonStringsTest {
    @Test
    fun readsClaimFields() {
        val raw = """{"token":"abc","device_id":"11111111-1111-1111-1111-111111111111"}"""
        assertEquals("abc", JsonStrings.string(raw, "token"))
        assertEquals("11111111-1111-1111-1111-111111111111", JsonStrings.string(raw, "device_id"))
    }
}
