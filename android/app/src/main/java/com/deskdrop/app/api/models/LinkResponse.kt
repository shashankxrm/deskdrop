package com.deskdrop.app.api.models

data class LinkResponse(
    val success: Boolean,
    val delivered: Boolean? = null,
    val queued: Boolean? = null,
    val message: String? = null
)

