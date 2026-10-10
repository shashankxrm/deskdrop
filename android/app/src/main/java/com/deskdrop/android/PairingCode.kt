package com.deskdrop.android

object PairingCode {
    fun normalize(raw: String): String {
        val b = StringBuilder()
        for (c in raw.trim().uppercase()) {
            when {
                c == '-' || c == ' ' -> Unit
                c in 'A'..'Z' || c in '0'..'9' -> b.append(c)
            }
        }
        if (b.length == 8) {
            return b.substring(0, 4) + "-" + b.substring(4)
        }
        return b.toString()
    }

    fun looksComplete(raw: String): Boolean = normalize(raw).length == 9 // ABCD-EFGH
}
