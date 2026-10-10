package com.deskdrop.android

import android.content.Context
import androidx.security.crypto.EncryptedSharedPreferences
import androidx.security.crypto.MasterKeys

class SessionStore(ctx: Context) {
    private val prefs = EncryptedSharedPreferences.create(
        "deskdrop_session",
        MasterKeys.getOrCreate(MasterKeys.AES256_GCM_SPEC),
        ctx,
        EncryptedSharedPreferences.PrefKeyEncryptionScheme.AES256_SIV,
        EncryptedSharedPreferences.PrefValueEncryptionScheme.AES256_GCM,
    )

    var baseUrl: String
        get() = prefs.getString(KEY_URL, "").orEmpty()
        set(v) { prefs.edit().putString(KEY_URL, v).apply() }

    val token: String get() = prefs.getString(KEY_TOKEN, "").orEmpty()
    val deviceId: String get() = prefs.getString(KEY_DEVICE, "").orEmpty()
    val isPaired: Boolean get() = token.isNotEmpty()

    fun saveClaim(baseUrl: String, token: String, deviceId: String) {
        prefs.edit()
            .putString(KEY_URL, baseUrl.trim().trimEnd('/'))
            .putString(KEY_TOKEN, token)
            .putString(KEY_DEVICE, deviceId)
            .apply()
    }

    fun clear() {
        prefs.edit().clear().apply()
    }

    companion object {
        private const val KEY_URL = "u"
        private const val KEY_TOKEN = "t"
        private const val KEY_DEVICE = "d"
    }
}
