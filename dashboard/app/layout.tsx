import type { Metadata } from "next";
import Link from "next/link";
import "./globals.css";

export const metadata: Metadata = {
  title: "deskdrop",
  description: "Pair devices, history, delete account.",
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <body>
        {children}
        <footer>
          <p>
            <Link href="/connect">Connect</Link>
            {" · "}
            <Link href="/privacy">Privacy</Link>
          </p>
        </footer>
      </body>
    </html>
  );
}