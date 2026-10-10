package com.deskdrop.android

import org.junit.Assert.assertEquals
import org.junit.Test
import java.io.IOException

class RetryingClaimTest {
    @Test
    fun retries429ThenSucceeds() {
        var n = 0
        val api = object : DeskdropApi() {
            override fun claim(baseUrl: String, code: String, name: String): ClaimResult {
                n++
                if (n < 3) return ClaimResult(429, null, null, "rate limited")
                return ClaimResult(201, "tok", "dev", null)
            }
        }
        val slept = mutableListOf<Long>()
        val res = RetryingClaim(
            api,
            Backoff(initialMs = 100, random = { 1.0 }),
            maxAttempts = 5,
            sleep = { slept.add(it) },
        ).claim("http://127.0.0.1:8080", "ABCD-EFGH")
        assertEquals(201, res.status)
        assertEquals("tok", res.token)
        assertEquals(3, n)
        assertEquals(listOf(100L, 200L), slept)
    }

    @Test
    fun doesNotRetry404() {
        var n = 0
        val api = object : DeskdropApi() {
            override fun claim(baseUrl: String, code: String, name: String): ClaimResult {
                n++
                return ClaimResult(404, null, null, "not found")
            }
        }
        val res = RetryingClaim(api, Backoff(), maxAttempts = 5, sleep = {}).claim("http://x", "ZZZZ-ZZZZ")
        assertEquals(404, res.status)
        assertEquals(1, n)
    }
}
