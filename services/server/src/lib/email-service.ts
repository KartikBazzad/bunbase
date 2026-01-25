/**
 * Email service for sending authentication emails
 * Supports email verification, password reset, and welcome emails
 * 
 * Currently uses console logging as a placeholder.
 * In production, integrate with SendGrid, AWS SES, Resend, or similar service.
 */

export interface EmailOptions {
  to: string;
  subject: string;
  html: string;
  text?: string;
}

export interface EmailVerificationData {
  email: string;
  verificationUrl: string;
  token: string;
}

export interface PasswordResetData {
  email: string;
  resetUrl: string;
  token: string;
  expiresIn: number; // minutes
}

/**
 * Send email verification email
 */
export async function sendVerificationEmail(
  data: EmailVerificationData,
): Promise<void> {
  const html = `
    <!DOCTYPE html>
    <html>
    <head>
      <meta charset="utf-8">
      <meta name="viewport" content="width=device-width, initial-scale=1.0">
      <title>Verify Your Email</title>
    </head>
    <body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333; max-width: 600px; margin: 0 auto; padding: 20px;">
      <div style="background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); padding: 30px; text-align: center; border-radius: 10px 10px 0 0;">
        <h1 style="color: white; margin: 0;">Verify Your Email</h1>
      </div>
      <div style="background: #f9f9f9; padding: 30px; border-radius: 0 0 10px 10px;">
        <p>Hello,</p>
        <p>Thank you for signing up for BunBase! Please verify your email address by clicking the button below:</p>
        <div style="text-align: center; margin: 30px 0;">
          <a href="${data.verificationUrl}" style="background: #667eea; color: white; padding: 12px 30px; text-decoration: none; border-radius: 5px; display: inline-block;">Verify Email</a>
        </div>
        <p>Or copy and paste this link into your browser:</p>
        <p style="word-break: break-all; color: #667eea;">${data.verificationUrl}</p>
        <p style="margin-top: 30px; font-size: 12px; color: #666;">This link will expire in 24 hours. If you didn't create an account, you can safely ignore this email.</p>
      </div>
    </body>
    </html>
  `;

  const text = `
    Verify Your Email
    
    Hello,
    
    Thank you for signing up for BunBase! Please verify your email address by visiting:
    
    ${data.verificationUrl}
    
    This link will expire in 24 hours. If you didn't create an account, you can safely ignore this email.
  `;

  await sendEmail({
    to: data.email,
    subject: "Verify Your Email Address",
    html,
    text,
  });
}

/**
 * Send password reset email
 */
export async function sendPasswordResetEmail(
  data: PasswordResetData,
): Promise<void> {
  const html = `
    <!DOCTYPE html>
    <html>
    <head>
      <meta charset="utf-8">
      <meta name="viewport" content="width=device-width, initial-scale=1.0">
      <title>Reset Your Password</title>
    </head>
    <body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333; max-width: 600px; margin: 0 auto; padding: 20px;">
      <div style="background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); padding: 30px; text-align: center; border-radius: 10px 10px 0 0;">
        <h1 style="color: white; margin: 0;">Reset Your Password</h1>
      </div>
      <div style="background: #f9f9f9; padding: 30px; border-radius: 0 0 10px 10px;">
        <p>Hello,</p>
        <p>We received a request to reset your password. Click the button below to create a new password:</p>
        <div style="text-align: center; margin: 30px 0;">
          <a href="${data.resetUrl}" style="background: #667eea; color: white; padding: 12px 30px; text-decoration: none; border-radius: 5px; display: inline-block;">Reset Password</a>
        </div>
        <p>Or copy and paste this link into your browser:</p>
        <p style="word-break: break-all; color: #667eea;">${data.resetUrl}</p>
        <p style="margin-top: 30px; font-size: 12px; color: #666;">This link will expire in ${data.expiresIn} minutes. If you didn't request a password reset, you can safely ignore this email.</p>
      </div>
    </body>
    </html>
  `;

  const text = `
    Reset Your Password
    
    Hello,
    
    We received a request to reset your password. Visit the link below to create a new password:
    
    ${data.resetUrl}
    
    This link will expire in ${data.expiresIn} minutes. If you didn't request a password reset, you can safely ignore this email.
  `;

  await sendEmail({
    to: data.email,
    subject: "Reset Your Password",
    html,
    text,
  });
}

/**
 * Send welcome email
 */
export async function sendWelcomeEmail(email: string, name: string): Promise<void> {
  const html = `
    <!DOCTYPE html>
    <html>
    <head>
      <meta charset="utf-8">
      <meta name="viewport" content="width=device-width, initial-scale=1.0">
      <title>Welcome to BunBase</title>
    </head>
    <body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333; max-width: 600px; margin: 0 auto; padding: 20px;">
      <div style="background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); padding: 30px; text-align: center; border-radius: 10px 10px 0 0;">
        <h1 style="color: white; margin: 0;">Welcome to BunBase!</h1>
      </div>
      <div style="background: #f9f9f9; padding: 30px; border-radius: 0 0 10px 10px;">
        <p>Hello ${name},</p>
        <p>Welcome to BunBase! We're excited to have you on board.</p>
        <p>Get started by creating your first project and exploring our powerful backend services.</p>
        <div style="text-align: center; margin: 30px 0;">
          <a href="${process.env.BETTER_AUTH_URL || "http://localhost:3000"}/dashboard" style="background: #667eea; color: white; padding: 12px 30px; text-decoration: none; border-radius: 5px; display: inline-block;">Go to Dashboard</a>
        </div>
        <p style="margin-top: 30px; font-size: 12px; color: #666;">If you have any questions, feel free to reach out to our support team.</p>
      </div>
    </body>
    </html>
  `;

  const text = `
    Welcome to BunBase!
    
    Hello ${name},
    
    Welcome to BunBase! We're excited to have you on board.
    
    Get started by visiting: ${process.env.BETTER_AUTH_URL || "http://localhost:3000"}/dashboard
    
    If you have any questions, feel free to reach out to our support team.
  `;

  await sendEmail({
    to: email,
    subject: "Welcome to BunBase!",
    html,
    text,
  });
}

/**
 * Core email sending function
 * In production, replace with actual email service integration
 */
async function sendEmail(options: EmailOptions): Promise<void> {
  // TODO: Integrate with email service provider (SendGrid, AWS SES, Resend, etc.)
  // For now, log to console in development
  if (process.env.NODE_ENV === "development") {
    console.log("📧 Email would be sent:", {
      to: options.to,
      subject: options.subject,
      // Don't log full HTML in production logs
      preview: options.text?.substring(0, 100) + "...",
    });

    // In development, you might want to actually send emails via a service
    // For production, uncomment and configure:
    /*
    if (process.env.EMAIL_SERVICE === "resend") {
      const { Resend } = await import("resend");
      const resend = new Resend(process.env.RESEND_API_KEY);
      await resend.emails.send({
        from: process.env.EMAIL_FROM!,
        to: options.to,
        subject: options.subject,
        html: options.html,
        text: options.text,
      });
    }
    */
  } else {
    // In production, integrate with your email service
    // Example with Resend:
    // const { Resend } = await import("resend");
    // const resend = new Resend(process.env.RESEND_API_KEY);
    // await resend.emails.send({ ... });
    
    throw new Error("Email service not configured. Please set up an email provider.");
  }
}
