package com.deskdrop.android

import java.net.URI

data class SharePayload(val url: String, val title: String)

object ShareUrl {
    const val maxUrlLen = 2048
    const val maxTitleLen = 512

    fun parse(raw: String): String? {
        val s = raw.trim()
        if (s.isEmpty() || s.length > maxUrlLen) return null
        if (s.any { it.isWhitespace() }) return null
        val u = try {
            URI(s)
        } catch (_: Exception) {
            return null
        }
        if (u.scheme != "http" && u.scheme != "https") return null
        if (u.host.isNullOrBlank()) return null
        return u.toString()
    }

    fun extract(raw: String): String? {
        parse(raw)?.let { return it }
        val m = Regex("""https?://[^\s<>\"]+""").find(raw) ?: return null
        val candidate = m.value.trimEnd(')', ']', ',', '.', '"', '\'', '>', ';')
        return parse(candidate)
    }

    fun fromSend(text: String, subject: String?): SharePayload? {
        val url = extract(text) ?: extract(subject.orEmpty()) ?: return null
        var title = subject?.trim().orEmpty()
        if (title == url || extract(title) == url) {
            title = ""
        }
        if (title.isEmpty()) {
            title = text.lineSequence()
                .map { it.trim() }
                .filter { it.isNotEmpty() && extract(it) == null }
                .joinToString(" ")
                .trim()
        }
        if (title.length > maxTitleLen) {
            title = title.take(maxTitleLen)
        }
        return SharePayload(url, title)
    }
}
