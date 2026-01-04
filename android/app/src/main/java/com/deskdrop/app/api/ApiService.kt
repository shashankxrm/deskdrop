package com.deskdrop.app.api

import com.deskdrop.app.api.models.LinkRequest
import com.deskdrop.app.api.models.LinkResponse
import retrofit2.Call
import retrofit2.http.Body
import retrofit2.http.POST

interface ApiService {
    @POST("links")
    fun submitLink(@Body request: LinkRequest): Call<LinkResponse>
}

