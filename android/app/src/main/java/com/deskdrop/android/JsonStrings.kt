package com.deskdrop.android

object JsonStrings {
    fun string(json: String, key: String): String? {
        val re = Regex(""""${Regex.escape(key)}"\s*:\s*"((?:\\.|[^"\\])*)"""")
        return re.find(json)?.groupValues?.get(1)?.replace("\\\"", "\"")
    }
}
