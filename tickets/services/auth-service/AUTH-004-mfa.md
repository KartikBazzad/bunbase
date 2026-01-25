# AUTH-004: Multi-Factor Authentication (MFA)

## Component
Authentication Service

## Type
Feature/Epic

## Priority
High

## Description
Implement multi-factor authentication (MFA) to add an extra layer of security. Support TOTP (Time-based One-Time Password), SMS-based 2FA, and backup codes. Provide enrollment, verification, and recovery flows.

## Requirements
Based on `requirements/auth-service.md` section 1.5

### Core Features
- TOTP (Time-based One-Time Password) support
- SMS-based 2FA
- Backup codes generation
- MFA enrollment flow
- MFA verification during login
- MFA recovery options
- Multiple MFA methods per user

## Technical Requirements

### API Endpoints
```
POST   /auth/mfa/enable           - Enable MFA for user
POST   /auth/mfa/disable          - Disable MFA for user
GET    /auth/mfa/status           - Get MFA status
POST   /auth/mfa/totp/setup       - Setup TOTP
POST   /auth/mfa/totp/verify      - Verify TOTP code
POST   /auth/mfa/sms/enable       - Enable SMS 2FA
POST   /auth/mfa/sms/verify       - Verify SMS code
POST   /auth/mfa/backup-codes/generate - Generate backup codes
POST   /auth/mfa/backup-codes/verify   - Verify backup code
POST   /auth/mfa/recover          - Recover account (MFA bypass)
```

### Database Schema
```typescript
interface MFASettings {
  id: string;
  userId: string;
  enabled: boolean;
  methods: string[]; // ['totp', 'sms']
  totpSecret?: string; // Encrypted
  totpVerified: boolean;
  smsPhoneNumber?: string;
  smsVerified: boolean;
  backupCodes?: string[]; // Hashed
  createdAt: Date;
  updatedAt: Date;
}

interface MFAAttempt {
  id: string;
  userId: string;
  method: string;
  success: boolean;
  ipAddress: string;
  userAgent: string;
  createdAt: Date;
}
```

### Performance Requirements
- TOTP code generation: < 50ms
- TOTP verification: < 100ms
- SMS OTP delivery: < 30 seconds
- Backup code verification: < 100ms

### Security Requirements
- TOTP secrets encrypted at rest
- Backup codes hashed before storage
- Maximum 5 failed MFA attempts before account lockout
- TOTP codes valid for 30-second windows
- SMS OTP codes expire in 10 minutes
- Backup codes are single-use
- MFA recovery requires additional verification

## Tasks

### 1. MFA Infrastructure
- [ ] Create MFASettings database schema
- [ ] Create MFAAttempt database schema
- [ ] Implement MFA status tracking
- [ ] Add MFA method enumeration
- [ ] Create MFA utilities

### 2. TOTP Implementation
- [ ] Integrate TOTP library (speakeasy/otplib)
- [ ] Implement TOTP secret generation
- [ ] Encrypt TOTP secrets
- [ ] Implement TOTP QR code generation
- [ ] Implement POST /auth/mfa/totp/setup endpoint
- [ ] Return QR code and secret for manual entry
- [ ] Implement POST /auth/mfa/totp/verify endpoint
- [ ] Validate TOTP code (30-second window)
- [ ] Handle clock skew tolerance
- [ ] Mark TOTP as verified on first use

### 3. SMS 2FA Implementation
- [ ] Implement POST /auth/mfa/sms/enable endpoint
- [ ] Validate phone number
- [ ] Send verification SMS
- [ ] Implement POST /auth/mfa/sms/verify endpoint
- [ ] Generate and send OTP code
- [ ] Verify OTP code
- [ ] Mark SMS 2FA as verified
- [ ] Integrate with SMS service

### 4. Backup Codes
- [ ] Implement backup code generation (10 codes)
- [ ] Hash backup codes before storage
- [ ] Implement POST /auth/mfa/backup-codes/generate endpoint
- [ ] Return codes to user (one-time display)
- [ ] Implement POST /auth/mfa/backup-codes/verify endpoint
- [ ] Verify backup code
- [ ] Mark backup code as used
- [ ] Prevent reuse of backup codes

### 5. MFA Enrollment Flow
- [ ] Implement POST /auth/mfa/enable endpoint
- [ ] Require password verification
- [ ] Allow selection of MFA method(s)
- [ ] Guide user through setup process
- [ ] Require verification of selected method(s)
- [ ] Generate backup codes on enrollment
- [ ] Update MFA status

### 6. MFA Verification During Login
- [ ] Modify login flow to check MFA status
- [ ] Require MFA verification if enabled
- [ ] Support TOTP verification
- [ ] Support SMS OTP verification
- [ ] Support backup code verification
- [ ] Track MFA attempts
- [ ] Lock account after failed attempts
- [ ] Return appropriate error codes

### 7. MFA Management
- [ ] Implement GET /auth/mfa/status endpoint
- [ ] Return enabled methods
- [ ] Return MFA enrollment status
- [ ] Implement POST /auth/mfa/disable endpoint
- [ ] Require password verification to disable
- [ ] Clear MFA settings on disable
- [ ] Log MFA disable events

### 8. MFA Recovery
- [ ] Implement POST /auth/mfa/recover endpoint
- [ ] Require email verification
- [ ] Require additional security questions (optional)
- [ ] Send recovery email with temporary bypass
- [ ] Allow temporary MFA bypass (24 hours)
- [ ] Log recovery attempts
- [ ] Notify user of recovery

### 9. Security Features
- [ ] Implement attempt tracking
- [ ] Add account lockout after failed attempts
- [ ] Encrypt sensitive data (TOTP secrets)
- [ ] Hash backup codes
- [ ] Add audit logging for MFA events
- [ ] Implement rate limiting for MFA endpoints

### 10. Error Handling
- [ ] Handle invalid TOTP codes
- [ ] Handle expired SMS OTPs
- [ ] Handle used backup codes
- [ ] Handle account lockout
- [ ] Create error codes for MFA flows

### 11. Testing
- [ ] Unit tests for TOTP generation/verification
- [ ] Unit tests for backup code generation
- [ ] Integration tests for TOTP enrollment
- [ ] Integration tests for SMS 2FA enrollment
- [ ] Integration tests for MFA verification
- [ ] Test account lockout
- [ ] Security tests

### 12. Documentation
- [ ] MFA setup guide
- [ ] TOTP configuration guide
- [ ] SMS 2FA setup guide
- [ ] Backup codes guide
- [ ] MFA recovery guide
- [ ] API endpoint documentation
- [ ] SDK integration examples

## Acceptance Criteria

- [ ] Users can enable TOTP MFA
- [ ] TOTP QR codes are generated correctly
- [ ] TOTP codes are verified successfully
- [ ] Users can enable SMS 2FA
- [ ] SMS OTP codes are sent and verified
- [ ] Backup codes are generated and work
- [ ] MFA is required during login if enabled
- [ ] Multiple MFA methods can be enabled
- [ ] Account lockout works after failed attempts
- [ ] MFA recovery flow works
- [ ] All security requirements are met
- [ ] Integration tests pass
- [ ] Documentation is complete

## Dependencies

- AUTH-001 (Email/Password Auth) - Login flow integration
- AUTH-003 (Magic Link/Phone Auth) - SMS service integration
- TOTP library (speakeasy/otplib)
- QR code generation library

## Estimated Effort
21 story points

## Related Requirements
- `requirements/auth-service.md` - Section 1.5
