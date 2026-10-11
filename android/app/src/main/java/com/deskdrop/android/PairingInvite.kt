package com.deskdrop.android

data class PairingInvite(val apiUrl: String?, val code: String)

object PairingInviteParse {
    private val urlRe = Regex("""(?i)(https?://[^\s/?#|]+)""")
    private val codeRe = Regex("""(?i)(?:[?&]code=|#)([A-Za-z0-9-]{4,16})""")

    fun parse(raw: String): PairingInvite {
        val s = raw.trim().trim('\uFEFF').replace("%7C", "|", ignoreCase = true)
        val api = urlRe.find(s)?.groupValues?.get(1)?.trimEnd('/')
        val fromQuery = codeRe.find(s)?.groupValues?.get(1)

        if (api != null) {
            val code = when {
                !fromQuery.isNullOrBlank() -> PairingCode.normalize(fromQuery)
                else -> pipeCode(s) ?: ""
            }
            return PairingInvite(api, code)
        }

        if (s.startsWith("deskdrop:", ignoreCase = true)) {
            val rest = s.substring("deskdrop:".length).trim()
            pipeCode(rest)?.let { code ->
                val left = rest.substring(0, rest.lastIndexOf('|')).trim().trimEnd('/')
                if (left.startsWith("http://", ignoreCase = true) || left.startsWith("https://", ignoreCase = true)) {
                    return PairingInvite(left, code)
                }
            }
            val only = PairingCode.normalize(rest)
            return PairingInvite(null, if (PairingCode.looksComplete(only)) only else "")
        }

        pipeCode(s)?.let { code ->
            val left = s.substring(0, s.lastIndexOf('|')).trim().trimEnd('/')
            if (left.startsWith("http://", ignoreCase = true) || left.startsWith("https://", ignoreCase = true)) {
                return PairingInvite(left, code)
            }
        }

        val only = PairingCode.normalize(s)
        if (PairingCode.looksComplete(only)) {
            return PairingInvite(null, only)
        }
        return PairingInvite(null, "")
    }

    private fun pipeCode(s: String): String? {
        val bar = s.lastIndexOf('|')
        if (bar <= 0) return null
        val right = PairingCode.normalize(s.substring(bar + 1))
        return right.takeIf { PairingCode.looksComplete(it) }
    }
}
