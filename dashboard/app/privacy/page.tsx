export default function PrivacyPage() {
  return (
    <main>
      <h1>Privacy</h1>
      <p>
        deskdrop identifies you as a pairing chain of devices, not an email or password. A device token is stored as
        HMAC-SHA256 on the server. The dashboard keeps that token in an httpOnly cookie, not in localStorage.
      </p>
      <ul>
        <li>We store the URL (and optional title) you share, until you ack or 30 days after delivery, then purge.</li>
        <li>We log request ids and JSON access logs. No advertising SDKs in this app.</li>
        <li>Delete account removes your user row and cascaded devices, codes, and links.</li>
        <li>Unpair revokes that device immediately; the next pull or page load is 401.</li>
      </ul>
      <p>
        <a href="/">Back</a>
      </p>
    </main>
  );
}
