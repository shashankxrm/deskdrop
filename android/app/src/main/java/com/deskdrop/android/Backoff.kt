package com.deskdrop.android

import kotlin.math.min
import kotlin.random.Random

/**
 * Full jitter backoff (AWS architecture blog): sleep in `[0, cap]` where
 * cap = min(maxMs, initialMs * 2^attempt).
 */
class Backoff(
    private val initialMs: Long = 500,
    private val maxMs: Long = 60_000,
    private val random: () -> Double = { Random.Default.nextDouble() },
) {
    fun delayMs(attempt: Int): Long {
        val exp = min(maxMs, initialMs * (1L shl attempt.coerceIn(0, 16)))
        val r = random().coerceIn(0.0, 1.0)
        return (exp * r).toLong()
    }

    companion object {
        fun shouldRetry(status: Int?, ioError: Boolean): Boolean {
            if (ioError) return true
            if (status == null) return false
            return status == 429 || status >= 500
        }
    }
}
