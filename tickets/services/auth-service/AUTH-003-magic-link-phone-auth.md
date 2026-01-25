# AUTH-003: Magic Link & Phone Authentication

## Component
Authentication Service

## Type
Feature/Epic

## Priority
Medium

## Description
Implement passwordless authentication methods including magic link (email-based) and phone number authentication with SMS OTP. These methods provide enhanced security and user experience for users who prefer not to use passwords.

## Requirements
Based on `requirements/auth-service.md` sections 1.3 and 1.4

### Core Features
- Magic link email authentication
- Time-limited magic links
- One-time use tokens
- SMS-based OTP authentication
- Phone number verification
- Rate limiting for SMS sends
- OTP code validation

## Technical Requirements

### API Endpoints
```
POST   /auth/magic-link/send      - Send magic link email
GET    /auth/magic-link/verify    - Verify magic link token
POST   /auth/phone/send-otp       - Send OTP to phone number
POST   /auth/phone/verify-otp     - Verify OTP code
POST   /auth/phone/verify-number  - Verify phone number ownership
```

### Database Schema
```typescript
interface MagicLinkToken {
  id: string;
  email: string;
  token: string; // Hashed
  expiresAt: Date;
  used: boolean;
  createdAt: Date;
}

interface PhoneOTP {
  id: string;
  phoneNumber: string;
  code: string; // Hashed
  expiresAt: Date;
  attempts: number;
  verified: boolean;
  createdAt: Date;
}
```

### Performance Requirements
- Magic link generation: < 100ms
- OTP generation: < 50ms
- SMS delivery: < 30 seconds (provider dependent)
- OTP verification: < 200ms

### Security Requirements
- Magic link tokens expire in 15 minutes
- OTP codes expire in 10 minutes
- OTP codes are 6 digits
- Rate limiting: 3 magic links per hour per email
- Rate limiting: 3 OTP sends per hour per phone
- OTP codes are hashed before storage
- Maximum 5 OTP verification attempts

## Tasks

### 1. Magic Link Infrastructure
- [ ] Create MagicLinkToken database schema
- [ ] Implement secure token generation
- [ ] Add token hashing utilities
- [ ] Implement token expiry logic
- [ ] Add one-time use validation

### 2. Magic Link Email Flow
- [ ] Implement POST /auth/magic-link/send endpoint
- [ ] Generate secure magic link token
- [ ] Create magic link URL
- [ ] Send magic link email
- [ ] Store token with expiry
- [ ] Add rate limiting (3 per hour)
- [ ] Implement GET /auth/magic-link/verify endpoint
- [ ] Validate token
- [ ] Check token expiry
- [ ] Mark token as used
- [ ] Create user session
- [ ] Handle invalid/expired tokens

### 3. Phone Authentication Infrastructure
- [ ] Create PhoneOTP database schema
- [ ] Implement OTP code generation (6 digits)
- [ ] Add OTP hashing utilities
- [ ] Implement OTP expiry logic
- [ ] Add attempt tracking

### 4. SMS Service Integration
- [ ] Integrate SMS provider (Twilio/AWS SNS)
- [ ] Create SMS template for OTP
- [ ] Implement SMS sending utilities
- [ ] Handle SMS delivery errors
- [ ] Add SMS delivery status tracking

### 5. Phone OTP Flow
- [ ] Implement POST /auth/phone/send-otp endpoint
- [ ] Validate phone number format
- [ ] Generate OTP code
- [ ] Hash and store OTP
- [ ] Send OTP via SMS
- [ ] Add rate limiting (3 per hour)
- [ ] Implement POST /auth/phone/verify-otp endpoint
- [ ] Validate OTP code
- [ ] Check OTP expiry
- [ ] Track verification attempts
- [ ] Create user account if new
- [ ] Create user session
- [ ] Mark OTP as verified
- [ ] Handle invalid/expired OTPs

### 6. Phone Number Verification
- [ ] Implement POST /auth/phone/verify-number endpoint
- [ ] Update phoneVerified flag
- [ ] Link phone to user account
- [ ] Handle phone number updates

### 7. Account Creation
- [ ] Auto-create user account on magic link verification
- [ ] Auto-create user account on OTP verification
- [ ] Link magic link/phone to existing account if email/phone matches
- [ ] Handle account merging scenarios

### 8. Rate Limiting
- [ ] Implement rate limiting for magic link sends
- [ ] Implement rate limiting for OTP sends
- [ ] Add IP-based rate limiting
- [ ] Add device fingerprinting
- [ ] Track and log rate limit violations

### 9. Error Handling
- [ ] Handle invalid phone numbers
- [ ] Handle SMS delivery failures
- [ ] Handle expired tokens/OTPs
- [ ] Handle exceeded verification attempts
- [ ] Create error codes for magic link/phone auth

### 10. Testing
- [ ] Unit tests for token/OTP generation
- [ ] Unit tests for validation logic
- [ ] Integration tests for magic link flow
- [ ] Integration tests for phone OTP flow
- [ ] Mock SMS provider for testing
- [ ] Test rate limiting
- [ ] Security tests

### 11. Documentation
- [ ] Magic link authentication guide
- [ ] Phone authentication guide
- [ ] API endpoint documentation
- [ ] SMS provider setup guide
- [ ] SDK integration examples

## Acceptance Criteria

- [ ] Magic link emails are sent successfully
- [ ] Magic link tokens expire after 15 minutes
- [ ] Magic link tokens are single-use
- [ ] Users can authenticate via magic link
- [ ] OTP codes are sent via SMS
- [ ] OTP codes expire after 10 minutes
- [ ] OTP verification works correctly
- [ ] Maximum 5 OTP verification attempts enforced
- [ ] Rate limiting prevents abuse
- [ ] Phone numbers can be verified
- [ ] User accounts are created automatically
- [ ] All security requirements are met
- [ ] Integration tests pass
- [ ] Documentation is complete

## Dependencies

- AUTH-001 (Email/Password Auth) - User management infrastructure
- Email service provider configured
- SMS service provider configured (Twilio/AWS SNS)

## Estimated Effort
13 story points

## Related Requirements
- `requirements/auth-service.md` - Sections 1.3, 1.4
