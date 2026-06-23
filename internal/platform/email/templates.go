package email

import "fmt"

func PasswordResetHTML(resetLink string) string {
	return fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
</head>
<body style="
  margin: 0;
  padding: 0;
  background-color: #091413;
  color: #dff2ea;
  font-family: 'Courier New', Courier, monospace;
  min-height: 100vh;
">
  <table width="100%%" cellpadding="0" cellspacing="0" style="padding: 48px 16px;">
    <tr>
      <td align="center">

        <!-- Card -->
        <table width="520" cellpadding="0" cellspacing="0" style="
          background-color: #0f1c1a;
          border: 1px solid #1e3530;
          border-radius: 12px;
          overflow: hidden;
        ">

          <!-- Header bar -->
          <tr>
            <td style="
              background-color: #162622;
              border-bottom: 1px solid #1e3530;
              padding: 20px 32px;
            ">
              <span style="
                color: #B0E4CC;
                font-size: 18px;
                font-weight: 700;
                letter-spacing: 0.05em;
              ">finai</span>
            </td>
          </tr>

          <!-- Body -->
          <tr>
            <td style="padding: 36px 32px;">

              <h2 style="
                margin: 0 0 12px 0;
                font-size: 20px;
                font-weight: 700;
                color: #dff2ea;
                letter-spacing: 0.02em;
              ">Reset your password</h2>

              <p style="
                margin: 0 0 28px 0;
                font-size: 14px;
                line-height: 1.6;
                color: #7aaa96;
              ">
                We received a request to reset the password for your account.
                Click the button below to continue. This link expires in
                <span style="color: #B0E4CC; font-weight: 600;">30 minutes</span>.
              </p>

              <!-- Button -->
              <table cellpadding="0" cellspacing="0">
                <tr>
                  <td style="
                    background-color: #408A71;
                    border-radius: 8px;
                  ">
                    <a href="%s" style="
                      display: inline-block;
                      padding: 12px 28px;
                      color: #dff2ea;
                      font-family: 'Courier New', Courier, monospace;
                      font-size: 14px;
                      font-weight: 600;
                      text-decoration: none;
                      letter-spacing: 0.03em;
                    ">Reset Password →</a>
                  </td>
                </tr>
              </table>

              <!-- Divider -->
              <table width="100%%" cellpadding="0" cellspacing="0" style="margin: 28px 0;">
                <tr>
                  <td style="
                    border-top: 1px solid #1e3530;
                    font-size: 0;
                    line-height: 0;
                  ">&nbsp;</td>
                </tr>
              </table>

              <!-- Link fallback -->
              <p style="
                margin: 0 0 6px 0;
                font-size: 12px;
                color: #3d6b5a;
              ">Or copy this link into your browser:</p>
              <p style="
                margin: 0 0 28px 0;
                font-size: 11px;
                color: #7aaa96;
                word-break: break-all;
                background-color: #162622;
                border: 1px solid #1e3530;
                border-radius: 6px;
                padding: 10px 12px;
              ">%s</p>

              <p style="
                margin: 0;
                font-size: 12px;
                color: #3d6b5a;
                line-height: 1.6;
              ">
                If you didn't request this, you can safely ignore this email.
                Your password will not be changed.
              </p>

            </td>
          </tr>

          <!-- Footer -->
          <tr>
            <td style="
              background-color: #162622;
              border-top: 1px solid #1e3530;
              padding: 16px 32px;
              text-align: center;
            ">
              <span style="
                font-size: 11px;
                color: #3d6b5a;
              ">© 2026 finai · You're receiving this because you requested a password reset</span>
            </td>
          </tr>

        </table>
        <!-- /Card -->

      </td>
    </tr>
  </table>
</body>
</html>`, resetLink, resetLink)
}
