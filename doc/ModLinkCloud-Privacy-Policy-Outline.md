# ModLinkCloud Privacy Policy — Outline

**Nature**: Outline for drafting the binding Privacy Policy (not a substitute for legal counsel)  
**Version**: 1.0  
**Last updated**: 2026-05-10  

---

## Important notice

1. Final text must reflect the **legal entity’s jurisdiction**, target markets (e.g., PIPL, GDPR), and actual data flows.  
2. Processing of **sensitive personal information** or **cross-border transfers** may require additional steps (notice, separate consent, assessments, SCCs, etc.) under applicable law.

---

## 1. Scope

This policy applies to ModLinkCloud websites, clients, APIs, admin console, and related services. Third-party sites linked from the product are governed by their own policies.

---

## 2. Data controller

- **Legal name**: To be filled  
- **Address**: To be filled  
- **Privacy contact**: Dedicated email or postal address — **to be filled**

---

## 3. Categories of data we may collect

### 3.1 You provide directly

| Category | Examples | Typical purposes |
|----------|----------|------------------|
| Account | Email, phone, username; password (stored hashed) | Registration, authentication |
| Business verification | Company name, registration number (if B2B) | Contracts, invoicing |
| Payments | Order IDs, transaction references from payment providers | **Full card numbers are typically handled by payment processors, not stored by us** (confirm implementation) |
| Support | Tickets, attachments you send | Customer support |

### 3.2 Generated automatically

| Category | Examples | Typical purposes |
|----------|----------|------------------|
| Device & logs | IP, device type, browser, coarse location if inferred, timestamps | Security, debugging, compliance |
| Usage & billing | Call counts, token usage, model IDs, request IDs | Billing, quotas, analytics |
| API metadata | Key identifiers (not the secret), routes | Authorization and abuse prevention |

### 3.3 AI-related content

If users submit **text, files, or images** to enable AI features, inputs and outputs may transit our systems and be forwarded to **third-party model providers**. Whether content is **persisted**, for how long, and who can access it must be stated precisely per product settings (including optional logging).

---

## 4. Purposes and legal bases

We process personal data to:

- Provide and improve the service (routing, billing, dashboards);  
- Authenticate users and protect against fraud and abuse;  
- Comply with law and respond to lawful requests;  
- Send **service-related** notices (marketing requires separate consent or opt-out where required);  
- Use aggregated or de-identified data for analytics.

Legal bases (contract, legitimate interests, consent, legal obligation) must be mapped **per jurisdiction** in the final policy.

---

## 5. Cookies and similar technologies

Describe cookies, local storage, analytics tools, retention, and how users can refuse or delete them.

---

## 6. Sharing, processors, and disclosure

### 6.1 Subprocessors

Cloud hosting, database, email/SMS, payments, support tooling, monitoring—list **categories or names** and constrain processing by contract.

### 6.2 Model providers (important)

AI inputs/outputs may be processed by **third-party inference providers**, potentially **outside your country**. Link to their privacy notices where practical and describe safeguards (contracts, minimization).

### 6.3 Legal and safety

Disclosure when required by law, court order, or to protect rights, safety, or security.

### 6.4 Business transfers

Notify users of successor obligations in mergers, acquisitions, or asset sales.

---

## 7. International transfers

If personal data leaves your country, describe **destination**, **safeguards** (e.g., SCCs, certification, statutory exemptions), and how users may obtain copies of relevant protections.

---

## 8. Retention

- **Account data**: For the life of the account and a limited period after deletion as required by law (e.g., tax).  
- **Logs**: Rolling retention window — **state number of days** once defined.  
- **Conversation / prompt storage**: Per user settings or default policy; clarify deletion on account closure and exceptions (legal hold).

---

## 9. Security

Summarize technical and organizational measures (encryption in transit, access control, least privilege). Avoid claiming “absolute security”; remind users to protect passwords and API keys.

---

## 10. Your rights

Depending on jurisdiction, users may have rights to **access, rectify, delete, restrict processing, object, data portability, withdraw consent**, and **account deletion**. Provide a **request channel** and **response timelines** aligned with local law.

---

## 11. Children

State minimum age, parental consent if applicable, or that the service is not directed at children.

---

## 12. Automated decision-making

If you use solely automated decisions with legal or similarly significant effects, disclose logic and user rights as required.

---

## 13. Changes to this policy

How updates are posted, when they take effect, and how material changes are notified.

---

## 14. Contact

Privacy inquiries — **to be filled**.

---

## Appendix — Admin visibility and logs

If admins can view **per-request logs, dashboards, or full prompts/responses**, disclose:

- Which roles see what;  
- Whether logs contain **full content** or metadata only;  
- Whether enterprise admins can access member data (tenant model).

---

**Drafting note**: Complete DPIA / transfer impact assessment where required; maintain a **subprocessor list** and **DPA** templates with enterprise customers.
