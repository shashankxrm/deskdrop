import type { Metadata } from "next";
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
            <a href="/privacy">Privacy</a>
          </p>
        </footer>
      </body>
    </html>
  );
}