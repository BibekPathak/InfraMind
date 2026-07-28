export default function RootLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return (
    <html lang="en">
      <body style={{ margin: 0, fontFamily: '-apple-system, BlinkMacSystemFont, sans-serif', background: '#0f172a', color: '#e2e8f0' }}>
        {children}
      </body>
    </html>
  )
}
