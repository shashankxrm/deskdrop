package com.deskdrop.android

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Test

class PairingInviteTest {
    @Test
    fun connectQuery() {
        val got = PairingInviteParse.parse("https://shashank-dropdesk.duckdns.org/connect?code=abcd-efgh")
        assertEquals("https://shashank-dropdesk.duckdns.org", got.apiUrl)
        assertEquals("ABCD-EFGH", got.code)
    }

    @Test
    fun qrWithOriginPipe() {
        val got = PairingInviteParse.parse("deskdrop:https://shashank-dropdesk.duckdns.org|abcd-efgh")
        assertEquals("https://shashank-dropdesk.duckdns.org", got.apiUrl)
        assertEquals("ABCD-EFGH", got.code)
    }

    @Test
    fun encodedPipe() {
        val got = PairingInviteParse.parse("deskdrop:https://shashank-dropdesk.duckdns.org%7Cabcd-efgh")
        assertEquals("https://shashank-dropdesk.duckdns.org", got.apiUrl)
        assertEquals("ABCD-EFGH", got.code)
    }

    @Test
    fun urlMustNotBecomeTheCode() {
        val got = PairingInviteParse.parse("https://shashank-dropdesk.duckdns.org")
        assertEquals("https://shashank-dropdesk.duckdns.org", got.apiUrl)
        assertEquals("", got.code)
    }

    @Test
    fun uppercaseUrlFromScanner() {
        val got =
            PairingInviteParse.parse("HTTPS://SHASHANK-DROPDESK.DUCKDNS.ORG/CONNECT?CODE=ABCD-EFGH")
        assertEquals("HTTPS://SHASHANK-DROPDESK.DUCKDNS.ORG", got.apiUrl)
        assertEquals("ABCD-EFGH", got.code)
    }

    @Test
    fun codeOnly() {
        val got = PairingInviteParse.parse("kz dn-sufr")
        assertNull(got.apiUrl)
        assertEquals("KZDN-SUFR", got.code)
    }
}
