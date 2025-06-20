# User Identity & Privacy: Technical Specifications

This document provides the detailed technical specifications for the User Identity & Privacy components of the DigiSocialBlock (Nexus Protocol). These specifications build upon the foundational DLI `EchoNet` protocol and translate the conceptual architecture from Phase 3 (Integrity, Privacy & Governance) and Phase 5 (Implementation Plan for Phase 3) into actionable details for implementation.

## 1. Decentralized Identity (DID) Management

### 1.1. DID Method Specification (Technical)

-   **Strategic Priority:** `Foundation-Critical`
-   **Key Concepts:** Self-sovereign identity, cryptography, W3C DID Core specifications, DID method syntax, DID document, verification methods, service endpoints.
-   **Why:** This specification defines the precise technical construction and resolution of Decentralized Identifiers (DIDs) within the DigiSocialBlock ecosystem, forming the bedrock of user sovereignty and verifiable identity.

#### 1.1.1. DID Method Name & Syntax:

*   **Method Name:** `did:echonet`
    *   **Rationale:** A custom DID method name signifies that DIDs are natively resolved using the DLI `EchoNet` infrastructure, providing clarity and context. *(KISS - Know Your Core: Clear, unambiguous naming linked to the platform.)*
*   **Method-Specific Identifier (MSI) Syntax:**
    *   The MSI will be a cryptographically generated unique identifier derived from the initial public key associated with the DID, ensuring global uniqueness and preventing collisions.
    *   **Format:** `did:echonet:<base58btc_encoded_multihash_of_initial_public_key>`
        *   Example: `did:echonet:zQ3shZc2QZ5m6rHn7Z...`
    *   **Rationale:**
        *   Deriving the MSI from the public key provides inherent proof of ownership at creation (controller is who generated the key).
        *   Using multihash allows for future cryptographic agility if underlying hash algorithms need to change. Base58btc encoding is common in blockchain/DID spaces for brevity and URL safety.
        *   *(KISS - Sense the Landscape: Cryptographically sound, collision-resistant, and future-proof identifier generation.)*

#### 1.1.2. Cryptographic Primitives:

*   **Public Key Cryptography:**
    *   **Standard Curve:** Ed25519 (Edwards-curve Digital Signature Algorithm over Curve25519) will be the primary recommended cryptographic suite for user DIDs.
        *   **Rationale:** Ed25519 offers high performance, excellent security, small key/signature sizes, and resistance to many side-channel attacks. It's widely supported.
    *   **Key Derivation:** Public keys will be directly used or derived as per standard Ed25519 practices.
*   **Hashing for MSI Generation:**
    *   **Algorithm:** SHA2-256 will be used to hash the initial public key. The result will then be encoded using Multihash (prefixing with hash algorithm code) and then Base58btc.
*   *(KISS - Sense the Landscape: Utilize strong, well-vetted, and widely adopted cryptographic standards.)*

#### 1.1.3. DID Document Structure & Content:

The DID Document for `did:echonet` DIDs will be a JSON-LD object adhering to the W3C DID Core specification.

*   **Core Properties:**
    *   `@context`: Must include `"https://www.w3.org/ns/did/v1"` and potentially `EchoNet`-specific context extensions.
    *   `id`: The `did:echonet:<MSI>` string.
    *   `verificationMethod`: An array of verification methods (public keys) associated with the DID.
        *   Each entry includes: `id` (e.g., `did:echonet:<MSI>#key-1`), `type` (e.g., `Ed25519VerificationKey2020` or a more current standard type), `controller` (the DID itself), `publicKeyMultibase` (the public key encoded in multibase format, typically base58btc for Ed25519).
    *   `authentication`: An array listing verification methods from `verificationMethod` authorized for authentication (e.g., signing challenges for login).
    *   `assertionMethod`: An array listing verification methods authorized for issuing Verifiable Credentials.
    *   `keyAgreement`: An array listing verification methods authorized for establishing encrypted communication channels (e.g., using X25519 keys derived from Ed25519 keys).
    *   `capabilityInvocation`: For authorizing actions (e.g., signing transactions on DLI `EchoNet`).
    *   `capabilityDelegation`: For delegating capabilities.
    *   *(KISS - Know Your Core: Adherence to W3C standards ensures interoperability and clarity of DID document structure.)*
*   **Optional Properties:**
    *   `service`: Array of service endpoints associated with the DID (e.g., social profile endpoints, communication service endpoints, DSN pointers for DID document extensions).
        *   Each service entry: `id`, `type`, `serviceEndpoint`.
    *   `created`: ISO8601 timestamp of DID registration.
    *   `updated`: ISO8601 timestamp of last DID document update.
    *   `alsoKnownAs`: URIs that identify the same DID subject.
*   **Storage of DID Document:**
    *   The full DID Document JSON-LD object will be stored off-chain, typically on the Distributed Data Stores (DDS) specified in `tech_specs/dli_echonet_protocol.md#2` or a user-chosen DSN (as per Phase 5 Decentralized Storage).
    *   The DLI `EchoNet` (via the On-System DID Registry - Sub-Issue 1.2) will store an immutable anchor: the `did:echonet:<MSI>` linked to the hash of its corresponding DID Document. This ensures DID document integrity.
    *   *(KISS - Systematize for Scalability: Keep on-chain DID data minimal (anchor and hash), store larger DID documents off-chain.)*

#### 1.1.4. DID Operations (CRUD - Create, Read, Update, Deactivate):

*   **Create (Register):**
    *   User generates an initial Ed25519 key pair locally.
    *   The MSI is derived from the public key.
    *   A minimal DID Document is constructed containing the `id` and the initial `verificationMethod` entry.
    *   The DID Document is stored on the DDS/DSN, yielding a content hash (or CID).
    *   The user (or their client) submits a transaction to the On-System DID Registry on DLI `EchoNet` to register the `did:echonet:<MSI>` with the hash of the DID Document. This transaction must be signed by the initial key.
*   **Read (Resolve):**
    *   See Sub-Issue 1.2 (On-System DID Registry & Resolver). The resolver fetches the DID Document hash from the DLI `EchoNet` registry, then retrieves the full DID Document from DDS/DSN using that hash.
*   **Update:**
    *   To update a DID Document (e.g., add/remove keys, update service endpoints), the DID controller:
        1.  Constructs the new DID Document.
        2.  Stores it on DDS/DSN, obtaining a new content hash.
        3.  Submits a transaction to the On-System DID Registry on DLI `EchoNet` to update the DID Document hash associated with their `did:echonet:<MSI>`. This transaction must be signed by a key authorized for `capabilityInvocation` or a specific update key defined in the current DID document.
    *   *(KISS - Iterate Intelligently: DID documents are versioned by their content hash; updates create new versions, maintaining history.)*
*   **Deactivate (Revoke):**
    *   The DID controller submits a transaction to the On-System DID Registry to mark the DID as deactivated. This typically involves setting a flag or replacing the DID Document hash with a tombstone record. The DID and its history may remain for auditability, but it's marked unusable. This must be signed by an authorized key.

#### Alignment with Expanded KISS Principles:

*   **Know Your Core, Keep it Clear:** The `did:echonet` method has a clear syntax and relies on precise DID document structures defined by W3C standards, ensuring unambiguous identity representation.
*   **Iterate Intelligently, Integrate Intuitively:** Adherence to W3C DID Core allows for integration with a broader ecosystem of DID tools and services. The method itself can be versioned if major changes are needed.
*   **Systematize for Scalability, Synchronize for Synergy:** Storing full DID documents off-chain (DDS/DSN) while anchoring hashes on DLI `EchoNet` ensures scalability. The DID method provides a common language for identity across all DigiSocialBlock services.
*   **Sense the Landscape, Secure the Solution:** Use of strong, standard cryptography (Ed25519, SHA-256). DID operations (update, deactivate) require authorization via cryptographic signatures from the DID controller. DID document integrity is ensured by on-chain hash anchoring.
*   **Stimulate Engagement, Sustain Impact:** A robust, user-controlled DID method is foundational for user trust, privacy, and data sovereignty, which are key to stimulating long-term engagement and ensuring the platform's positive impact.

This DID Method Specification provides the technical DNA for self-sovereign identity within the DigiSocialBlock (Nexus Protocol) ecosystem.

### 1.2. On-System DID Registry & Resolver (DLI `EchoNet` Integration)

-   **Strategic Priority:** `Foundation-Critical`
-   **Estimated Effort:** `Large`
-   **Key Concepts:** DID registration, DID resolution, DLI `EchoNet` record types, DDS integration, cryptographic verification, data integrity.
-   **Why:** To provide a concrete, on-system mechanism for anchoring `did:echonet` identifiers to their corresponding DID Document hashes and enabling authoritative resolution of these DIDs. This makes DIDs practically usable within the DigiSocialBlock ecosystem.

#### 1.2.1. Core Objectives:

*   **Authoritative Registration:** Securely record the association between a `did:echonet` identifier and the hash of its DID Document on the DLI `EchoNet`.
*   **Integrity:** Ensure that the registered DID Document hash cannot be tampered with once recorded.
*   **Efficient Resolution:** Allow any party to efficiently look up the DID Document hash associated with a given `did:echonet` identifier.
*   **Controlled Updates:** Ensure only the DID controller can update the associated DID Document hash.

#### 1.2.2. DID Registry on DLI `EchoNet`:

The DLI `EchoNet` itself will serve as the ledger for DID registration. This will not be a separate smart contract in the traditional sense but rather a dedicated record type or state transition logic within the core DLI `EchoNet` protocol, managed by Super-Hosts and Decelerators through the established multi-step validation flow (from Phase 1 architecture).

*   **`DIDRegistryAnchorRecord` (DLI `EchoNet` Record Type):**
    *   **Structure (Conceptual - to be part of DLI `EchoNet`'s native record types):**
        *   `record_type`: Fixed value indicating "DID_REGISTRY_ANCHOR".
        *   `did_msi`: The Method-Specific Identifier string (e.g., `zQ3shZc2QZ5m6rHn7Z...`). This is the primary key for lookups.
        *   `did_document_hash`: The SHA2-256 hash of the canonicalized DID Document (JSON-LD) that is stored on the DDS/DSN.
        *   `controller_did_msi`: The MSI of the DID controller (initially the DID itself, can be updated).
        *   `status`: Enum (e.g., `ACTIVE`, `DEACTIVATED`).
        *   `version_id`: Incremental version number for the record, updated with each change.
        *   `timestamp`: DLI `EchoNet` Network Witnessed Timestamp of the last update to this anchor record.
        *   `signature`: Cryptographic signature from the `controller_did_msi` authorizing the registration or update.
    *   *(KISS - Know Your Core: A lean on-system record, primarily linking the DID MSI to the off-chain document's hash and controller.)*
    *   **Storage:** These `DIDRegistryAnchorRecord`s will be part of the DLI `EchoNet`'s state, replicated across Super-Hosts' Active Storage and eventually part of the Block Archive. They will be indexed by `did_msi` for efficient lookups.

*   **Operations (as DLI `EchoNet` Transactions):**
    1.  **`RegisterDIDAnchor (did_msi, did_document_hash, controller_did_msi, signature)`:**
        *   A specific DLI `EchoNet` transaction type.
        *   **Validation (by Super-Hosts, then Decelerators):**
            *   Verify the `signature` using the public key associated with `controller_did_msi` (resolved via this same DID system, creating a slight bootstrap consideration for the very first DIDs or relying on `did:key` for initial controllers).
            *   Ensure `did_msi` is not already registered.
            *   Validate format of `did_msi` and `did_document_hash`.
        *   If valid, a new `DIDRegistryAnchorRecord` is created and committed to the DLI `EchoNet` state.
        *   *(KISS - Sense the Landscape: Strong cryptographic control over DID registration.)*
    2.  **`UpdateDIDAnchor (did_msi, new_did_document_hash, new_controller_did_msi (optional), signature)`:**
        *   A specific DLI `EchoNet` transaction type.
        *   **Validation:**
            *   Verify `signature` against the *current* `controller_did_msi` of the existing `DIDRegistryAnchorRecord` for the given `did_msi`.
            *   Updates `did_document_hash`, optionally `controller_did_msi`, increments `version_id`, updates `timestamp`.
    3.  **`DeactivateDIDAnchor (did_msi, signature)`:**
        *   A specific DLI `EchoNet` transaction type.
        *   **Validation:** Verify `signature` against the current `controller_did_msi`.
        *   Sets `status` to `DEACTIVATED`.

#### 1.2.3. DID Resolver Mechanism (DLI `EchoNet` Query):

DID Resolution is the process of taking a DID URI and producing a compliant DID Document.

*   **Resolution Steps:**
    1.  **Client Request:** A client (e.g., mobile app, backend service) wishes to resolve a `did:echonet:<MSI>`.
    2.  **Query DLI `EchoNet` Registry:** The client (or a resolver service it uses) queries the DLI `EchoNet` (specifically, Super-Hosts' Active Storage for recent DIDs, or deeper storage for older ones) for the `DIDRegistryAnchorRecord` associated with the `<MSI>`.
        *   This query would be a standard DLI `EchoNet` read operation, not necessarily a transaction.
    3.  **Retrieve Anchor Record:** If the DID is registered and active, the DLI `EchoNet` returns the `DIDRegistryAnchorRecord` containing the `did_document_hash`.
    4.  **Fetch DID Document from DDS/DSN:** The client then uses the `did_document_hash` (which is effectively the CID or DSN reference) to retrieve the full DID Document (JSON-LD object) from the Distributed Data Stores (DDS) or the specified DSN (as per Section 2 of this document and Phase 5 Decentralized Storage).
    5.  **Verify DID Document Integrity:** The client re-hashes the retrieved DID Document and compares it against the `did_document_hash` from the on-system anchor record. If they match, the DID Document is considered authentic.
    *   *(KISS - Know Your Core: Clear separation of concerns: DLI `EchoNet` for anchor integrity, DDS/DSN for document storage.)*

*   **Resolver Implementation:**
    *   This can be implemented as a library within the DigiSocialBlock SDKs (Component 5.2 of Phase 5).
    *   Alternatively, dedicated resolver services (potentially run by Decelerators or community nodes) could offer a public API endpoint for DID resolution.

#### Alignment with Expanded KISS Principles:

*   **Know Your Core, Keep it Clear:** The registry stores minimal, essential data on DLI `EchoNet` (DID MSI -> Document Hash, Controller). The resolution process is clearly defined.
*   **Iterate Intelligently, Integrate Intuitively:** The resolver can be a simple library initially. More advanced features like caching resolvers or universal resolver integration can be added later. DID Document updates create new versions, allowing history tracking.
*   **Systematize for Scalability, Synchronize for Synergy:** Storing full documents off-chain on DDS/DSN is key for scalability. The DLI `EchoNet` registry ensures all nodes have a synchronized view of the latest valid DID document hashes. The resolver synergizes the DLI `EchoNet` registry with the DDS/DSN.
*   **Sense the Landscape, Secure the Solution:** All state changes to the DID registry (register, update, deactivate) are DLI `EchoNet` transactions, secured by the DID controller's signature and validated by the network's multi-step validation process. Integrity of the DID document is ensured by comparing its hash with the on-system anchor.
*   **Stimulate Engagement, Sustain Impact:** A reliable and secure DID registry and resolver are fundamental for any application or user interaction requiring trusted identity, thereby enabling a wide range of engaging features.

#### Dependencies & Challenges:

*   **Dependencies:**
    *   `1.1. DID Method Specification`: Defines the DIDs being registered/resolved.
    *   DLI `EchoNet` Core Protocol: Must support the `DIDRegistryAnchorRecord` type and associated transaction validation logic.
    *   Distributed Data Stores (DDS) / Decentralized Storage Networks (DSN): For storing the actual DID Documents.
    *   Reliable DLI `EchoNet` query mechanism for clients/resolvers.
*   **Challenges:**
    *   **Bootstrap/Genesis DIDs:** How are the very first DIDs (e.g., for initial Leadership Council members or system services) registered before the full system is live? (Potential solution: pre-configure in genesis state).
    *   **DID Document Availability on DDS/DSN:** While the anchor is on DLI `EchoNet`, the resolution process depends on the DID Document being available on the external storage layer. Pinning strategies (Phase 5) for DID documents are important.
    *   **Resolver Complexity & Performance:** Ensuring resolvers are efficient and can quickly fetch data from both DLI `EchoNet` and DDS/DSN. Caching strategies for resolved DID documents will be important.
    *   **Key Rotation & Controller Updates:** The `UpdateDIDAnchor` operation must securely handle changes to authorized keys or even the controller of the DID, which is a sensitive operation.

This On-System DID Registry and Resolver mechanism provides the practical machinery for making `did:echonet` DIDs a functional and trustworthy cornerstone of the DigiSocialBlock identity layer.

### 1.3. Initial DID Creation & Management (User Flow & API)

-   **Strategic Priority:** `High`
-   **Estimated Effort:** `Medium`
-   **Key Concepts:** User onboarding, key generation, DID document bootstrapping, API design, secure key storage (client-side), DID update transactions.
-   **Why:** To define the practical user experience and technical pathways for users to create, secure, and manage their `did:echonet` DIDs, making self-sovereign identity accessible and usable within DigiSocialBlock.

#### 1.3.1. Core Objectives:

*   **User-Friendly Creation:** Enable users to easily create a new `did:echonet` DID, ideally with minimal technical friction.
*   **Secure Key Management:** Emphasize secure generation and storage of private keys on the user's device.
*   **DID Document Bootstrapping:** Define how the initial DID Document is created and anchored to the DLI `EchoNet`.
*   **Controlled Updates:** Provide mechanisms for users to manage their DID Document (e.g., key rotation, adding service endpoints) via secure, authorized transactions.

#### 1.3.2. Initial DID Creation User Flow (Conceptual):

This flow describes how a new user creates their `did:echonet` DID, typically via the DigiSocialBlock mobile application (Host client).

1.  **Initiation:** User selects "Create New Identity" (or similar) in the app.
2.  **Key Pair Generation:**
    *   The client application generates a new Ed25519 key pair locally on the user's device.
    *   The private key MUST be stored securely, ideally in the device's hardware-backed keystore (e.g., Android Keystore, iOS Secure Enclave). User is warned about the importance of backing up their recovery phrase/private key.
    *   *(KISS - Sense the Landscape: Security of private keys is paramount from the very start.)*
3.  **DID & MSI Derivation:**
    *   The `did:echonet` Method-Specific Identifier (MSI) is derived from the newly generated public key as per specification 1.1.2. The full DID URI is formed.
4.  **Initial DID Document Creation:**
    *   A minimal, compliant DID Document (JSON-LD) is constructed locally by the client. It includes:
        *   `@context`
        *   `id` (the new `did:echonet:<MSI>`)
        *   Initial `verificationMethod` entry using the generated public key (e.g., for `authentication`, `assertionMethod`, `capabilityInvocation`).
        *   `controller` (set to the DID itself).
    *   *(KISS - Know Your Core: Start with a minimal, valid DID document representing core control.)*
5.  **DID Document Storage (DDS/DSN):**
    *   The client application uploads this initial DID Document to the designated DDS/DSN (as per Phase 5 Decentralized Storage and Section 2 of `dli_echonet_protocol.md`).
    *   The DDS/DSN returns the content hash (CID) of the stored DID Document.
6.  **On-System Registration Transaction:**
    *   The client application constructs a `RegisterDIDAnchor` transaction for the DLI `EchoNet` (as per spec 1.2.2). This transaction includes:
        *   `did_msi`
        *   `did_document_hash` (the CID from step 5)
        *   `controller_did_msi` (the `did_msi` itself for initial registration)
    *   This transaction is signed using the newly generated private key.
7.  **Transaction Submission & Confirmation:**
    *   The signed transaction is submitted to the user's Cell Super-Host and proceeds through the DLI `EchoNet` validation and commitment process.
    *   The client UI provides feedback to the user on the submission status (pending, confirmed, failed). *(KISS - Iterate Intelligently: Clear user feedback during asynchronous operations.)*
8.  **Post-Creation:** User is prompted to securely back up their recovery phrase / private key.

#### 1.3.3. DID Management API Endpoints (Conceptual - for Client-Backend Interaction):

While many DID operations are direct DLI `EchoNet` transactions signed client-side, some backend assistance or standardized API flows might be useful for abstraction or integration with other services (e.g., `AuthService` from Phase 3 Implementation Plan). These are conceptual endpoints that a backend service (e.g., a "DID Helper Service" or integrated within `AuthService`) might expose, which in turn would help clients construct and submit DLI `EchoNet` transactions.

*   **`POST /did/create_registration_transaction` (Helper Endpoint):**
    *   **Request:** `{ initial_public_key_multibase, did_document_hash_from_dds }`
    *   **Response:** A pre-filled, unsigned `RegisterDIDAnchor` DLI `EchoNet` transaction structure that the client can then sign and submit.
    *   **Rationale:** Simplifies client-side construction of the initial registration transaction. Backend can validate inputs.
*   **`POST /did/create_update_transaction` (Helper Endpoint):**
    *   **Request:** `{ did_msi_to_update, new_did_document_hash_from_dds, new_controller_did_msi (optional) }`
    *   **Response:** A pre-filled, unsigned `UpdateDIDAnchor` DLI `EchoNet` transaction structure. Client signs this with a currently authorized key for the DID.
*   **`POST /did/create_deactivate_transaction` (Helper Endpoint):**
    *   **Request:** `{ did_msi_to_deactivate }`
    *   **Response:** A pre-filled, unsigned `DeactivateDIDAnchor` DLI `EchoNet` transaction structure. Client signs with an authorized key.
*   **`GET /did/resolve/{did_uri}` (Passthrough to DLI `EchoNet` Resolver):**
    *   A convenience endpoint that uses the server-side DLI `EchoNet` resolver (spec 1.2.3) to fetch and return a DID Document. Caching can be implemented here.
    *   *(KISS - Systematize for Scalability: Server-side resolution with caching can reduce load on individual clients needing to perform full resolution.)*

#### 1.3.4. Key Management & Rotation (User Responsibility, Client-Facilitated):

*   **Key Rotation:**
    *   Users must be able to rotate keys associated with their DID (e.g., if a key is compromised or for proactive security).
    *   **Flow:**
        1.  User generates a new key pair on their device.
        2.  User constructs an updated DID Document that adds the new public key to `verificationMethod` and potentially updates `authentication`, `assertionMethod`, etc., to use the new key. The old key might be deprecated or removed (depending on policy).
        3.  User uploads the new DID Document to DDS/DSN, gets new hash.
        4.  User (via client) creates an `UpdateDIDAnchor` transaction, signed by a *current, authorized key* for the DID, pointing to the new DID Document hash.
*   **Device Recovery / Adding New Devices:**
    *   Relies on the user having securely backed up their original private key(s) or recovery phrase.
    *   A new device can re-import the private key to gain control of the DID.
    *   Social recovery mechanisms or multi-sig control by trusted DIDs (future iteration) could enhance this. *(KISS - Iterate Intelligently: Start with user-managed backup, explore advanced recovery later.)*

#### Alignment with Expanded KISS Principles:

*   **Know Your Core, Keep it Clear:** The user flow for DID creation is broken down into clear, logical steps. APIs have clear purposes. The core is empowering users to create and manage their own DIDs.
*   **Iterate Intelligently, Integrate Intuitively:** The initial creation flow is straightforward. Key rotation and more advanced management features build upon this. Helper API endpoints aim to make integration intuitive for client developers.
*   **Systematize for Scalability, Synchronize for Synergy:** While creation is user-driven, the helper APIs and resolver endpoints can be scaled as backend services. DID management events on DLI `EchoNet` allow other services to synchronize.
*   **Sense the Landscape, Secure the Solution:** Strong emphasis on secure local key generation and storage. All DID management operations that change state are authorized by cryptographic signatures. Users are educated on backup responsibilities.
*   **Stimulate Engagement, Sustain Impact:** A simple and secure DID creation/management process is crucial for user adoption of decentralized identity, which underpins many of DigiSocialBlock's unique features and fosters user trust and sovereignty.

#### Dependencies & Challenges:

*   **Dependencies:**
    *   `1.1. DID Method Specification`: Defines the DIDs being created.
    *   `1.2. On-System DID Registry & Resolver`: The DLI `EchoNet` mechanisms these user flows interact with.
    *   Client-side cryptographic libraries for key generation and signing.
    *   Secure storage capabilities on user devices (hardware keystores).
    *   DDS/DSN integration for storing DID documents.
*   **Challenges:**
    *   **User Experience (UX) of Key Management:** This is a notorious challenge in Web3. Making private key generation, storage, and backup intuitive and secure for non-technical users is paramount. "Seedless" or social recovery options are complex future considerations.
    *   **Educating Users:** Users need to understand the responsibility that comes with self-sovereign keys.
    *   **Interoperability of Backups:** Ensuring recovery phrases/private keys can be easily imported/exported across different client applications or wallets adhering to standards.
    *   **API Security:** If helper API endpoints are provided, they must be secured against abuse.
    *   **Transaction Costs:** DID registration and updates are DLI `EchoNet` transactions and will incur fees. These must be minimized for user adoption.

This specification for initial DID creation and management provides the practical pathways for users to claim and control their self-sovereign identities on DigiSocialBlock.

## 2. On-System Data Consent Protocol

### 2.1. Consent Record Data Model (DLI `EchoNet` Data Structure)

-   **Strategic Priority:** `Foundation-Critical`
-   **Estimated Effort:** `Medium`
-   **Key Concepts:** Data privacy, consent granularity, immutability, auditability, DLI `EchoNet` native record types.
-   **Why:** To define the precise, immutable, and verifiable data structure for recording user consent on the DLI `EchoNet`, forming the auditable backbone of DigiSocialBlock's data sovereignty and privacy commitments.

#### 2.1.1. Core Objectives:

*   **Clarity & Unambiguity:** The structure must clearly define who consented, to what service/actor, for what data, for what purpose, and for how long.
*   **Verifiability:** Consent records must be linkable to the user's DID and verifiable via their signature.
*   **Immutability (of Record):** Once a consent record (or its revocation) is committed to the DLI `EchoNet`, it cannot be altered. Updates are new records or state changes.
*   **Auditability:** The history of consent grants and revocations should be traceable.

#### 2.1.2. `NexusConsentRecord` (DLI `EchoNet` Native Record Type):

This structure defines a single instance of user consent, recorded on the DLI `EchoNet`. It will be a native record type, processed and validated by the DLI `EchoNet`'s core logic (Super-Hosts, Decelerators).

```
// Conceptual representation (Protobuf-like or language-agnostic for DLI EchoNet native record)
message NexusConsentRecord {
  // --- Core Consent Identifiers ---
  string consent_record_id = 1;         // Primary Identifier: Unique hash of (data_subject_did + data_controller_did + purpose_hash + scope_hash + issuance_timestamp_nonce)
  string data_subject_did = 2;          // DID of the user granting consent. (KISS - Know Your Core: Clear subject)
  string data_controller_did = 3;       // DID of the service, DApp, or user requesting/receiving consent. (KISS - Know Your Core: Clear controller)

  // --- Consent Specifics ---
  bytes32 purpose_hash = 4;             // Hash of a canonicalized, human-readable (and potentially machine-readable coded) description of the data processing purpose. Full text stored off-chain/DDS, referenced by this hash.
                                        // Example purposes: "PersonalizedFeed_v1", "Analytics_Anonymized_v1", "ThirdPartyApp_CoolDApp_DataShare_v1.2"
  bytes32 scope_hash = 5;               // Hash of a canonicalized description of the data scope (categories of data) being consented to. Full text stored off-chain/DDS.
                                        // Example scopes: "BasicProfileData", "InteractionMetrics_LikesOnly", "DirectMessages_ReadWrite_WithUserXYZ"
  string specific_policy_version_ref = 6; // Reference (e.g., hash or URI) to the specific version of a human-readable privacy policy or data usage agreement that this consent pertains to. (KISS - Sense the Landscape: Links consent to clear terms)

  // --- Temporal & Status ---
  uint64 consent_granted_timestamp = 7;   // DLI EchoNet Network Witnessed Timestamp of when the consent grant transaction was committed.
  uint64 consent_expiry_timestamp = 8;    // Unix timestamp (UTC). 0 if indefinite until explicitly revoked.
  ConsentStatus current_status = 9;     // Enum: GRANTED, REVOKED, EXPIRED.
  uint64 status_update_timestamp = 10;  // DLI EchoNet Network Witnessed Timestamp of the last status change (grant, revoke, expire).
  uint32 version = 11;                  // Version of this consent record (e.g., if scope/expiry can be updated, though revocation + new grant is often cleaner).

  // --- Proof & Auditability ---
  bytes subject_signature_on_grant = 12; // Signature of data_subject_did over (data_controller_did + purpose_hash + scope_hash + specific_policy_version_ref + consent_granted_timestamp_client_asserted_nonce)
                                         // This signature is part of the transaction that creates this record.
  // Revocation would be a separate transaction type referencing this consent_record_id, also signed by subject.
}

enum ConsentStatus {
  UNDEFINED_CONSENT_STATUS = 0;
  GRANTED = 1;
  REVOKED_BY_SUBJECT = 2;
  EXPIRED_AUTOMATICALLY = 3;
  SUPERSEDED_BY_NEW_CONSENT = 4; // If updates create new records
}

```
*   **Off-Chain Storage of Full Text:** The full human-readable (and potentially machine-readable) text for `purpose_description`, `scope_description`, and `specific_policy_version_ref` will be stored on the DDS/DSN. Their hashes are stored on-chain in the `NexusConsentRecord` to ensure integrity and keep the on-system record lean.
    *   *(KISS - Systematize for Scalability: Minimize on-system storage, ensure verifiability via hashes.)*
*   **Indexing:** `NexusConsentRecord`s should be queryable/indexable on the DLI `EchoNet` (primarily via Active Storage managed by Super-Hosts) by:
    *   `data_subject_did` (for users to see their consents)
    *   `data_controller_did` (for services to verify consents given to them)
    *   `consent_record_id` (for direct lookup)

#### Alignment with Expanded KISS Principles:

*   **Know Your Core, Keep it Clear:** The `NexusConsentRecord` precisely defines the "who, what, why, to whom, and for how long" of data consent. Hashing detailed descriptions keeps the on-system record clear and focused on verifiable anchors.
*   **Iterate Intelligently, Integrate Intuitively:** The use of `specific_policy_version_ref` allows policies to evolve; new consents would reference new policy versions. The `version` field on the record itself allows for future evolution of the consent structure if needed, though aiming for immutability of individual grant/revoke actions is preferred.
*   **Systematize for Scalability, Synchronize for Synergy:** Storing full policy texts off-chain on DDS/DSN is critical for DLI `EchoNet` scalability. On-system indexing by subject and controller DID allows relevant services to efficiently query and synchronize with user consent states.
*   **Sense the Landscape, Secure the Solution:**
    *   User signatures (`subject_signature_on_grant`) provide cryptographic proof of consent.
    *   Immutability of committed records on DLI `EchoNet` ensures an auditable trail.
    *   Hashing of off-chain policy/scope/purpose texts ensures the terms of consent cannot be changed after the fact without detection.
*   **Stimulate Engagement, Sustain Impact:** Transparent, user-controlled, and verifiable consent is fundamental to building user trust and encouraging engagement with platform features that may require data access, knowing they have control. This directly sustains the platform's integrity and ethical standing.

#### Dependencies & Challenges:

*   **Dependencies:**
    *   DLI `EchoNet` Core Protocol: Must support this native record type and its associated transaction validation logic.
    *   Section 1: Decentralized Identity (DID) Management: For `data_subject_did` and `data_controller_did`.
    *   Distributed Data Stores (DDS) / Decentralized Storage Networks (DSN): For storing full text of purposes, scopes, and policies.
    *   Content Hashing & Timestamping (Section 4 of DLI EchoNet specs): For `purpose_hash`, `scope_hash`, and secure timestamps.
*   **Challenges:**
    *   **Canonicalization of Purpose/Scope for Hashing:** Defining a strict canonical format for purpose and scope descriptions before hashing them is crucial for consistent verification.
    *   **UX for Presenting Hashed Details:** While hashes ensure integrity, the UI must seamlessly resolve these hashes to human-readable descriptions from DDS/DSN when presenting consent information to users or controllers.
    *   **Granularity of Consent:** Defining appropriate levels of granularity for "purpose" and "scope" that are meaningful to users but also manageable for developers and services. Too granular can lead to consent fatigue; too broad can undermine user control.
    *   **Revocation Complexity:** Ensuring revocations are processed reliably and that all relevant services are promptly notified or can verify the updated status (link to Sub-Issue 2.2 - Consent Enforcement).
    *   **Initial Policy Definitions:** Bootstrapping the system with clear, fair, and comprehensive initial versions of privacy policies and standard purpose/scope descriptions.

This `NexusConsentRecord` data model provides the immutable, verifiable foundation for on-system data consent within DigiSocialBlock.

### 2.2. Consent Granting & Revocation Protocol

-   **Strategic Priority:** `High`
-   **Estimated Effort:** `Large`
-   **Key Concepts:** DLI `EchoNet` transaction types, cryptographic signatures, user intent, event propagation, state changes, consent lifecycle.
-   **Why:** To define the specific on-system protocols and transaction structures that enable users to actively and verifiably grant and revoke consent for data processing, ensuring their control over data is actionable and auditable.

#### 2.2.1. Core Objectives:

*   **Actionability:** Provide users with concrete actions (transactions) to express their consent decisions.
*   **Verifiability & Non-Repudiation:** Ensure consent grants and revocations are signed by the user's DID and immutably recorded on the DLI `EchoNet`.
*   **Atomicity (for initial grant):** The initial grant of consent should ideally be a single, clear transaction.
*   **Timeliness:** Consent status changes should be reflected on the DLI `EchoNet` promptly.

#### 2.2.2. DLI `EchoNet` Transaction Types for Consent:

Two primary DLI `EchoNet` transaction types will be defined to manage the lifecycle of a `NexusConsentRecord`. These transactions will be processed by the DLI `EchoNet`'s standard validation flow (Super-Hosts, Decelerators, Leadership Council).

1.  **`GrantConsentTransaction`:**
    *   **Purpose:** To create a new `NexusConsentRecord` on the DLI `EchoNet`, signifying the user's explicit granting of consent.
    *   **Payload Fields (Conceptual):**
        *   `transaction_type`: Fixed value "GRANT_CONSENT".
        *   `data_controller_did`: The DID of the service/DApp requesting consent.
        *   `purpose_hash`: Hash of the purpose description (as defined in 2.1.2).
        *   `scope_hash`: Hash of the scope description (as defined in 2.1.2).
        *   `specific_policy_version_ref`: Reference to the policy version being consented to.
        *   `consent_expiry_timestamp_requested`: Requested expiry (Unix timestamp, UTC; 0 for indefinite). User client calculates this based on duration.
        *   `client_asserted_issuance_nonce`: A client-generated nonce or timestamp to contribute to the uniqueness of the `consent_record_id` hash and the `subject_signature_on_grant`.
        *   `data_subject_signature`: Cryptographic signature (from `data_subject_did` which is the transaction sender's DID) over `(data_controller_did + purpose_hash + scope_hash + specific_policy_version_ref + client_asserted_issuance_nonce)`. This becomes `NexusConsentRecord.subject_signature_on_grant`.
    *   **Processing Logic by DLI `EchoNet` Nodes (Super-Hosts/Decelerators):**
        1.  Verify `data_subject_signature` against the sender's DID. (*KISS - Sense the Landscape: Authentication of intent*)
        2.  Validate field formats (DIDs, hashes, timestamp format).
        3.  Generate the unique `consent_record_id` (e.g., hash of `data_subject_did + data_controller_did + purpose_hash + scope_hash + client_asserted_issuance_nonce`). Check for collisions (though astronomically unlikely with good hashing).
        4.  Create a new `NexusConsentRecord` (as per 2.1.2) with:
            *   Populated fields from the transaction.
            *   `consent_granted_timestamp` set to the Network Witnessed Timestamp of this transaction's commitment.
            *   `current_status` set to `GRANTED`.
            *   `status_update_timestamp` set to `consent_granted_timestamp`.
            *   `version` set to 1.
        5.  Commit the new `NexusConsentRecord` to DLI `EchoNet` state.
        6.  Emit an on-system event (e.g., `ConsentGrantedEvent(consent_record_id, data_subject_did, data_controller_did)`).
    *   *(KISS - Know Your Core: A single, clear transaction type for granting consent.)*

2.  **`RevokeConsentTransaction`:**
    *   **Purpose:** To change the status of an existing `NexusConsentRecord` to `REVOKED_BY_SUBJECT`.
    *   **Payload Fields (Conceptual):**
        *   `transaction_type`: Fixed value "REVOKE_CONSENT".
        *   `consent_record_id_to_revoke`: The ID of the existing `NexusConsentRecord` to be revoked.
        *   `data_subject_signature`: Cryptographic signature (from `data_subject_did`) over `consent_record_id_to_revoke` and a current timestamp/nonce to prevent replay.
    *   **Processing Logic by DLI `EchoNet` Nodes:**
        1.  Verify `data_subject_signature` against the sender's DID.
        2.  Retrieve the existing `NexusConsentRecord` using `consent_record_id_to_revoke`.
        3.  Verify that the sender's DID matches the `data_subject_did` in the stored record. (*KISS - Sense the Landscape: Only the data subject can revoke their consent.*)
        4.  Verify the record is currently in a `GRANTED` status.
        5.  Update the `NexusConsentRecord`:
            *   Set `current_status` to `REVOKED_BY_SUBJECT`.
            *   Set `status_update_timestamp` to the Network Witnessed Timestamp of this transaction's commitment.
            *   Increment `version`.
        6.  Commit the updated `NexusConsentRecord` to DLI `EchoNet` state.
        7.  Emit an on-system event (e.g., `ConsentRevokedEvent(consent_record_id, data_subject_did, data_controller_did)`).
    *   *(KISS - Know Your Core: A specific transaction for revocation, ensuring clear intent and auditability.)*

#### 2.2.3. Consent Update (Conceptual):

*   While a direct "update" transaction for an existing consent record (e.g., to change scope or expiry) could be defined, the cleaner and more auditable approach, often preferred for consent, is:
    1.  User initiates a `RevokeConsentTransaction` for the old consent.
    2.  User initiates a new `GrantConsentTransaction` with the modified terms.
*   This ensures that each set of consent terms has its own distinct, signed record and clear grant/expiry timestamps. It avoids ambiguity about which terms were active when.
*   *(KISS - Know Your Core & Iterate Intelligently: Favoring explicit revocation and re-grant for clarity and auditability, even if it means two transactions from a user perspective, which can be bundled by the client UI.)*

#### Alignment with Expanded KISS Principles:

*   **Know Your Core, Keep it Clear:** Distinct transaction types for `GrantConsent` and `RevokeConsent` make user intent explicit and the protocol logic straightforward.
*   **Iterate Intelligently, Integrate Intuitively:** The protocol uses standard transaction patterns. Client applications will abstract the transaction creation, making it intuitive for users (e.g., a simple "Allow" / "Deny" button that triggers these transactions).
*   **Systematize for Scalability, Synchronize for Synergy:** These DLI `EchoNet` transactions are processed by the existing scalable node infrastructure. Emitted events allow backend enforcement services to synchronize with consent status changes.
*   **Sense the Landscape, Secure the Solution:** Cryptographic signatures ensure authenticity and non-repudiation of consent actions. Only the data subject can grant or revoke their specific consent records. Replay attacks on revocations are mitigated by including a timestamp/nonce in the signature.
*   **Stimulate Engagement, Sustain Impact:** A clear, secure, and user-driven protocol for managing consent empowers users, builds trust, and encourages them to engage with services that require data access, knowing they have ultimate control. This is fundamental to the ethical operation and sustained impact of DigiSocialBlock.

#### Dependencies & Challenges:

*   **Dependencies:**
    *   `2.1. Consent Record Data Model`: Defines the state being manipulated by these transactions.
    *   DLI `EchoNet` Core Protocol: For transaction submission, validation, commitment, and event emission.
    *   Section 1: Decentralized Identity (DID) Management: For identifying and authenticating the `data_subject_did`.
    *   Client-side applications (e.g., mobile app) to construct, sign, and submit these transactions based on user UI interactions.
*   **Challenges:**
    *   **User Experience (UX):** Abstracting the transaction-based nature of consent management into a seamless and understandable UI. Users should not feel like they are "sending crypto transactions" for every consent change.
    *   **Transaction Fees:** Each consent grant/revocation is a DLI `EchoNet` transaction and will incur fees. These must be kept minimal to avoid discouraging users from managing their consent actively. (Consideration: Could the `data_controller` sponsor fees for `GrantConsent` in some scenarios?)
    *   **State Management & Querying:** Efficiently querying the current state of all consents for a user or for a data controller (Sub-Issue 2.3 will deal with enforcement logic which relies on this).
    *   **Eventual Consistency & Enforcement Lag:** There might be a small delay between a user revoking consent on DLI `EchoNet` and all data processing services becoming aware of this revocation. Enforcement logic (2.3) needs to handle this.

This protocol provides the active mechanisms for users to exercise their data sovereignty on DigiSocialBlock.

### 2.3. Consent Enforcement Logic (Backend/Service Level)

-   **Strategic Priority:** `High`
-   **Estimated Effort:** `Large`
-   **Key Concepts:** Access control, privacy engineering, data flow management, consent verification, caching strategies, real-time checks, default-deny.
-   **Why:** To define the technical mechanisms by which backend services and DApps within the DigiSocialBlock ecosystem actively check and adhere to user-defined consents recorded on the DLI `EchoNet`. This translates the on-system consent records into actual data protection and privacy enforcement.

#### 2.3.1. Core Objectives:

*   **Strict Adherence:** Ensure data processing only occurs if valid, active consent exists for the specific user, controller, purpose, and scope.
*   **Real-time (or Near Real-time) Verification:** Enable services to check the latest consent status before accessing or processing data.
*   **Default-Deny:** If consent is not explicitly granted, or is revoked/expired, access/processing must be denied by default.
*   **Auditability of Checks (Conceptual):** Log consent check events (success/failure) for internal audit and debugging (without logging user data itself).

#### 2.3.2. `ConsentService` (Backend Service - Technical Specification):

As conceptualized in the Phase 3 Implementation Plan, a dedicated `ConsentService` (or a distributed library/middleware with equivalent functionality) will be crucial.

*   **Primary Responsibilities:**
    1.  **Interface with DLI `EchoNet`:** Securely query the DLI `EchoNet` (Super-Host Active Storage or deeper archives) for `NexusConsentRecord`s.
    2.  **Cache Consent States:** Maintain a cache of recently accessed/verified consent records to optimize performance and reduce direct DLI `EchoNet` load. The cache must have a clear invalidation strategy based on on-system events (`ConsentGrantedEvent`, `ConsentRevokedEvent`) or Time-To-Live (TTL). *(KISS - Systematize for Scalability: Caching is vital for performance.)*
    3.  **Provide Verification API:** Offer a secure, internal API for other backend services/DApps to check consent status.

*   **Internal API Endpoints (Conceptual):**
    *   `POST /consent/verify`
        *   **Request Body:** `{ data_subject_did: string, data_controller_did: string, purpose_hash: bytes32, scope_hash: bytes32 }`
        *   **Response:**
            *   Success (200 OK): `{ consent_active: true, expiry_timestamp: uint64, consent_record_id: string }`
            *   No Consent / Revoked / Expired (200 OK or 403 Forbidden based on policy): `{ consent_active: false, reason: string (e.g., "NO_RECORD_FOUND", "REVOKED", "EXPIRED") }`
            *   Error (500 Internal Server Error): `{ error: string }`
        *   **Logic:**
            1.  Check local cache for the specific consent parameters.
            2.  If not in cache or stale, query DLI `EchoNet` for the `NexusConsentRecord` (using indices on `data_subject_did`, `data_controller_did`, `purpose_hash`, `scope_hash`).
            3.  Verify `current_status` is `GRANTED` and `consent_expiry_timestamp` (if not 0) is in the future.
            4.  Update cache. Return result.
    *   *(KISS - Know Your Core: A clear, specific API for consent verification.)*

*   **Event Subscription:** The `ConsentService` should subscribe to `ConsentGrantedEvent` and `ConsentRevokedEvent` emitted by the DLI `EchoNet` (via the consent protocol transactions) to proactively update its cache or invalidate relevant entries.

#### 2.3.3. Integration Pattern for Data-Processing Services:

All backend services or DApp backends within the DigiSocialBlock ecosystem that process user data covered by the consent protocol MUST implement the following pattern before any such data processing:

1.  **Identify Context:** Determine `data_subject_did` (from authenticated session), `data_controller_did` (the service itself or the entity it represents), `purpose_hash` (of the intended processing), and `scope_hash` (of the data to be processed).
2.  **Call `ConsentService.verify`:** Make a request to the `ConsentService` with these parameters.
3.  **Enforce Decision:**
    *   If `consent_active: true`, proceed with data processing.
    *   If `consent_active: false`, **DO NOT** process the data. Log the denial (for audit) and return an appropriate error or default non-personalized experience to the requesting function/user.
    *   *(KISS - Sense the Landscape: Default-deny is a critical security and privacy principle.)*
4.  **Error Handling:** Gracefully handle errors from the `ConsentService` (e.g., temporary unavailability) by defaulting to a privacy-preserving state (i.e., assume no consent).

#### 2.3.4. Frontend Considerations (UI Feedback):

*   While the core enforcement is backend, frontend UIs (from Phase 2) must:
    *   Clearly request consent before features that require it are activated.
    *   Provide feedback if an action cannot be completed because necessary consent is missing or has been revoked.
    *   Direct users to the Consent Management Dashboard (specified in Phase 3 Implementation Plan 2.2) if they need to grant or modify consents.

#### Alignment with Expanded KISS Principles:

*   **Know Your Core, Keep it Clear:** The enforcement logic is simple: check consent, then act. The `ConsentService` has a clear role. `purpose_hash` and `scope_hash` ensure clarity on what is being checked.
*   **Iterate Intelligently, Integrate Intuitively:** The `ConsentService` API provides a clear integration point. Initial enforcement can be strict; future iterations might explore more nuanced handling for specific contexts (e.g., "essential service" consents vs. "optional feature" consents), always governed by user choice.
*   **Systematize for Scalability, Synchronize for Synergy:** The `ConsentService` (with caching and event-driven updates) is designed to allow many other services to check consent efficiently. This ensures all data-processing components are synchronized with the user's expressed wishes.
*   **Sense the Landscape, Secure the Solution:** The default-deny principle is paramount. The system is designed to fail safe (i.e., deny access if consent cannot be confirmed). Logging of consent checks (not the data itself) aids in auditing and debugging potential privacy breaches or enforcement failures.
*   **Stimulate Engagement, Sustain Impact:** Knowing that their consent choices are actively enforced builds immense user trust. This encourages users to explore and use platform features more confidently, leading to deeper engagement and belief in the platform's ethical stance, which is critical for long-term impact.

#### Dependencies & Challenges:

*   **Dependencies:**
    *   `2.1. Consent Record Data Model`: The structure of the records being checked.
    *   `2.2. Consent Granting & Revocation Protocol`: The on-system transactions that create and update these records and emit events.
    *   DLI `EchoNet` query capabilities and event subscription mechanism.
    *   `1.2. DID-Based Authentication & Authorization`: To securely identify the `data_subject_did`.
    *   Clear definitions of `purpose_hash` and `scope_hash` used consistently across the ecosystem.
*   **Challenges:**
    *   **Performance & Latency:** Ensuring that consent checks (even with caching) do not introduce unacceptable latency into data processing workflows.
    *   **Cache Coherency:** Keeping the `ConsentService` cache accurately synchronized with on-system state, especially during high volumes of consent changes or network partitions.
    *   **Developer Compliance:** Ensuring *all* relevant services rigorously implement and respect the consent checks. This requires strong development guidelines, code review practices, and potentially automated testing.
    *   **Granularity of Checks:** If consents are highly granular, services might need to perform multiple checks for a single user operation, adding complexity.
    *   **Error Propagation & User Feedback:** Clearly communicating to users *why* an action failed due to consent, without revealing unnecessary internal details.
    *   **Auditing Enforcement:** Developing tools or methods to audit and prove that enforcement logic is working correctly across all services.

This Consent Enforcement Logic is the active shield that protects user data based on their expressed preferences, making data sovereignty a practical reality in DigiSocialBlock.

### 3. Unit Testing Strategy for User Identity & Privacy

-   **Strategic Priority:** `Foundation-Critical`
-   **Estimated Effort:** `Epic` (reflecting the thoroughness required for these core security and privacy components)
-   **Key Concepts:** Test-Driven Development (TDD) principles, cryptographic test vectors, mock objects, edge case analysis, positive/negative testing, code coverage, CI/CD integration.
-   **Why:** To guarantee the reliability, integrity, and security of the Decentralized Identity (DID) Management and On-System Data Consent Protocol components. Comprehensive unit testing is non-negotiable for these foundational elements that underpin user trust and data sovereignty in DigiSocialBlock.

#### 3.1. Core Objectives:

*   **Correctness:** Verify that each function, method, and module within the DID and Consent protocols behaves exactly as specified.
*   **Security:** Test for vulnerabilities related to cryptographic operations, authorization logic, and input validation.
*   **Robustness:** Ensure components handle edge cases, invalid inputs, and unexpected states gracefully.
*   **Isolation:** Test individual units of code (functions, classes, methods) in isolation to pinpoint failures accurately.
*   **Regression Prevention:** Create a comprehensive suite of tests that can be run automatically to prevent regressions as the codebase evolves.
*   *(KISS - Sense the Landscape, Secure the Solution: Rigorous testing is a primary defense against vulnerabilities and operational failures.)*

#### 3.2. Scope of Unit Testing:

Unit tests will cover all newly implemented code for:

1.  **Decentralized Identity (DID) Management (Section 1):**
    *   **1.1. DID Method Specification:**
        *   Test MSI generation logic from public keys.
        *   Test DID Document construction and validation against JSON-LD schemas and W3C DID Core compliance rules (where applicable locally).
        *   Test cryptographic primitive interactions (key generation, hashing for MSI) using known test vectors where possible.
    *   **1.2. On-System DID Registry & Resolver (DLI `EchoNet` Integration):**
        *   Test the logic within DLI `EchoNet` nodes (Super-Hosts/Decelerators) for processing `RegisterDIDAnchor`, `UpdateDIDAnchor`, and `DeactivateDIDAnchor` transactions:
            *   Signature verification logic.
            *   Validation of input fields.
            *   Correct creation and updating of `DIDRegistryAnchorRecord` state.
            *   Correct handling of already registered DIDs, unauthorized updates, etc. (negative tests).
        *   Test the DLI `EchoNet` query logic for DID resolution (fetching `DIDRegistryAnchorRecord`).
    *   **1.3. Initial DID Creation & Management (User Flow & API - focusing on backend helper logic if any):**
        *   Test any backend helper API endpoint logic for constructing unsigned transactions (e.g., `/did/create_registration_transaction`).
        *   Test validation logic within these helper APIs.

2.  **On-System Data Consent Protocol (Section 2):**
    *   **2.1. Consent Record Data Model:**
        *   Test `NexusConsentRecord` creation logic, ensuring all fields are correctly populated and validated.
        *   Test unique `consent_record_id` generation.
        *   Test logic for hashing off-chain purpose/scope descriptions.
    *   **2.2. Consent Granting & Revocation Protocol:**
        *   Test the DLI `EchoNet` node logic for processing `GrantConsentTransaction` and `RevokeConsentTransaction`:
            *   Signature verification.
            *   Validation of all input fields.
            *   Correct creation/update of `NexusConsentRecord` state (status, timestamps, version).
            *   Correct emission of `ConsentGrantedEvent` and `ConsentRevokedEvent`.
            *   Handling of invalid requests (e.g., revoking non-existent consent, revoking consent not owned by sender).
    *   **2.3. Consent Enforcement Logic (Backend/Service Level):**
        *   Test the `ConsentService` internal API (`/consent/verify`):
            *   Correctly queries DLI `EchoNet` or cache for consent records.
            *   Accurately interprets `NexusConsentRecord` status and expiry.
            *   Returns correct `consent_active` status under various conditions (granted, revoked, expired, no record).
            *   Test caching logic: cache hits, misses, invalidation upon DLI `EchoNet` events.
        *   Test integration points in mock data-processing services to ensure they correctly call `ConsentService.verify` and adhere to the default-deny principle.

#### 3.3. Testing Methodologies & Tools:

*   **Language-Specific Unit Testing Frameworks:**
    *   Utilize standard unit testing frameworks for the chosen implementation language(s) (e.g., Go's `testing` package, Rust's `#[test]` framework, Jest/Mocha for TypeScript/JavaScript, PyTest for Python).
*   **Mocking & Dependency Injection:**
    *   Heavily use mocking for external dependencies, especially:
        *   DLI `EchoNet` state/queries: Mock the interface for reading/writing DLI records so that DID/Consent logic can be tested without needing a running DLI network.
        *   DDS/DSN interactions: Mock interfaces for storing/retrieving DID documents or full consent policy texts.
        *   Cryptography: While core crypto libraries are trusted, interfaces to them can be mocked for specific scenarios (e.g., simulating signature verification failure).
    *   Employ dependency injection to make components easily testable with mock dependencies.
    *   *(KISS - Iterate Intelligently: Testable code through modularity and mocking.)*
*   **Test Data & Scenarios:**
    *   Develop a comprehensive set of test data, including valid inputs, invalid inputs, edge cases (e.g., empty strings, max length strings, zero values, future/past timestamps).
    *   Create specific test scenarios for each function/method covering:
        *   **Positive tests:** Ensuring correct behavior with valid inputs.
        *   **Negative tests:** Ensuring graceful error handling and rejection of invalid inputs or unauthorized actions.
        *   **Boundary value analysis.**
*   **Cryptographic Test Vectors:**
    *   Where applicable (e.g., signature schemes, hashing), use known cryptographic test vectors to ensure correctness of underlying crypto operations.
*   **Code Coverage:**
    *   Aim for high unit test code coverage (e.g., >80-90% for critical modules) as measured by standard code coverage tools. While not a perfect metric, it indicates thoroughness.

#### 3.4. Integration with CI/CD Pipeline:

*   All unit tests MUST be integrated into the Continuous Integration/Continuous Delivery (CI/CD) pipeline (as conceptualized in Phase 4).
*   Builds should fail if any unit tests fail.
*   Regularly run the full suite of unit tests to catch regressions quickly.
*   *(KISS - Iterate Intelligently: Automated testing in CI/CD is fundamental for reliable, continuous development.)*

#### Alignment with Expanded KISS Principles:

*   **Know Your Core, Keep it Clear:** Each unit test focuses on a single, clear piece of functionality or requirement for the DID and Consent protocols. Test names should clearly describe what they are testing.
*   **Iterate Intelligently, Integrate Intuitively:** A strong unit test suite allows developers to refactor and iterate on code with confidence, knowing that regressions will be caught. TDD principles can be applied.
*   **Systematize for Scalability, Synchronize for Synergy:** While unit tests focus on individual components, their collective success ensures that these components can later be integrated reliably into the larger, scalable system.
*   **Sense the Landscape, Secure the Solution:** This is the primary focus. Unit tests for DID and Consent will rigorously probe for security flaws, incorrect authorization logic, and data integrity issues. Testing cryptographic interactions is key.
*   **Stimulate Engagement, Sustain Impact:** High-quality, well-tested code builds trust – trust from developers working on the system, and ultimately trust from users who rely on its security and privacy features. This is essential for sustained impact.

#### Dependencies & Challenges:

*   **Dependencies:**
    *   Finalized technical specifications for all DID Management and On-System Data Consent Protocol components (Sections 1 and 2 of this document).
    *   Chosen implementation language(s) and their respective unit testing frameworks/tools.
    *   CI/CD infrastructure.
*   **Challenges:**
    *   **Thoroughness:** Ensuring all critical paths, edge cases, and security assumptions are adequately tested. This requires meticulous test case design.
    *   **Mocking Complexity:** Mocking interactions with the DLI `EchoNet` state or external DSNs can be complex but is necessary for true unit isolation.
    *   **Maintaining Test Suite:** As the protocol evolves, the unit test suite must be diligently maintained and updated. Outdated tests are worse than no tests.
    *   **Testing Cryptography:** Correctly testing cryptographic code requires specialized knowledge and careful use of test vectors. It's easy to write tests that provide a false sense of security if not done right.
    *   **Effort Estimation:** Writing comprehensive unit tests, especially for security-critical components, is a significant time investment but one that pays off in the long run. The "Epic" effort estimation reflects this.

A comprehensive and rigorously executed Unit Testing Strategy is the bedrock upon which the security, reliability, and trustworthiness of DigiSocialBlock's User Identity & Privacy layer will be built.
