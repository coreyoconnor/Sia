package types

// transaction.go defines the transaction type and all of the sub-fields of the
// transaction, as well as providing helper functions for working with
// transactions. The various IDs are designed such that, in a legal blockchain,
// it is cryptographically unlikely that any two objects would share an id.

import (
	"errors"
	"io"

	"github.com/NebulousLabs/Sia/crypto"
	"github.com/NebulousLabs/Sia/encoding"
)

const (
	SpecifierLen = 16

	// UnlockHashChecksumSize is the size of the checksum used to verify
	// human-readable addresses. It is not a crypytographically secure
	// checksum, it's merely intended to prevent typos. 6 is chosen because it
	// brings the total size of the address to 38 bytes, leaving 2 bytes for
	// potential version additions in the future.
	UnlockHashChecksumSize = 6
)

// These Specifiers are used internally when calculating a type's ID. See
// Specifier for more details.
var (
	SpecifierMinerPayout          = Specifier{'m', 'i', 'n', 'e', 'r', ' ', 'p', 'a', 'y', 'o', 'u', 't'}
	SpecifierSiacoinInput         = Specifier{'s', 'i', 'a', 'c', 'o', 'i', 'n', ' ', 'i', 'n', 'p', 'u', 't'}
	SpecifierSiacoinOutput        = Specifier{'s', 'i', 'a', 'c', 'o', 'i', 'n', ' ', 'o', 'u', 't', 'p', 'u', 't'}
	SpecifierFileContract         = Specifier{'f', 'i', 'l', 'e', ' ', 'c', 'o', 'n', 't', 'r', 'a', 'c', 't'}
	SpecifierFileContractRevision = Specifier{'f', 'i', 'l', 'e', ' ', 'c', 'o', 'n', 't', 'r', 'a', 'c', 't', ' ', 'r', 'e'}
	SpecifierStorageProof         = Specifier{'s', 't', 'o', 'r', 'a', 'g', 'e', ' ', 'p', 'r', 'o', 'o', 'f'}
	SpecifierStorageProofOutput   = Specifier{'s', 't', 'o', 'r', 'a', 'g', 'e', ' ', 'p', 'r', 'o', 'o', 'f'}
	SpecifierSiafundInput         = Specifier{'s', 'i', 'a', 'f', 'u', 'n', 'd', ' ', 'i', 'n', 'p', 'u', 't'}
	SpecifierSiafundOutput        = Specifier{'s', 'i', 'a', 'f', 'u', 'n', 'd', ' ', 'o', 'u', 't', 'p', 'u', 't'}
	SpecifierClaimOutput          = Specifier{'c', 'l', 'a', 'i', 'm', ' ', 'o', 'u', 't', 'p', 'u', 't'}
	SpecifierMinerFee             = Specifier{'m', 'i', 'n', 'e', 'r', ' ', 'f', 'e', 'e'}

	ErrTransactionIDWrongLen = errors.New("input has wrong length to be an encoded transaction id")
)

type (
	// A Specifier is a fixed-length byte-array that serves two purposes. In
	// the wire protocol, they are used to identify a particular encoding
	// algorithm, signature algorithm, etc. This allows nodes to communicate on
	// their own terms; for example, to reduce bandwidth costs, a node might
	// only accept compressed messages.
	//
	// Internally, Specifiers are used to guarantee unique IDs. Various
	// consensus types have an associated ID, calculated by hashing the data
	// contained in the type. By prepending the data with Specifier, we can
	// guarantee that distinct types will never produce the same hash.
	Specifier [SpecifierLen]byte

	// IDs are used to refer to a type without revealing its contents. They
	// are constructed by hashing specific fields of the type, along with a
	// Specifier. While all of these types are hashes, defining type aliases
	// gives us type safety and makes the code more readable.
	TransactionID   crypto.Hash
	SiacoinOutputID crypto.Hash
	SiafundOutputID crypto.Hash
	FileContractID  crypto.Hash
	OutputID        crypto.Hash

	// A Transaction is an atomic component of a block. Transactions can contain
	// inputs and outputs, file contracts, storage proofs, and even arbitrary
	// data. They can also contain signatures to prove that a given party has
	// approved the transaction, or at least a particular subset of it.
	//
	// Transactions can depend on other previous transactions in the same block,
	// but transactions cannot spend outputs that they create or otherwise be
	// self-dependent.
	Transaction struct {
		SiacoinInputs         []SiacoinInput         `json:"siacoininputs"`
		SiacoinOutputs        []SiacoinOutput        `json:"siacoinoutputs"`
		FileContracts         []FileContract         `json:"filecontracts"`
		FileContractRevisions []FileContractRevision `json:"filecontractrevisions"`
		StorageProofs         []StorageProof         `json:"storageproofs"`
		SiafundInputs         []SiafundInput         `json:"siafundinputs"`
		SiafundOutputs        []SiafundOutput        `json:"siafundoutputs"`
		MinerFees             []Currency             `json:"minerfees"`
		ArbitraryData         [][]byte               `json:"arbitrarydata"`
		TransactionSignatures []TransactionSignature `json:"transactionsignatures"`
	}

	// A SiacoinInput consumes a SiacoinOutput and adds the siacoins to the set of
	// siacoins that can be spent in the transaction. The ParentID points to the
	// output that is getting consumed, and the UnlockConditions contain the rules
	// for spending the output. The UnlockConditions must match the UnlockHash of
	// the output.
	SiacoinInput struct {
		ParentID         SiacoinOutputID  `json:"parentid"`
		UnlockConditions UnlockConditions `json:"unlockconditions"`
	}

	// A SiacoinOutput holds a volume of siacoins. Outputs must be spent
	// atomically; that is, they must all be spent in the same transaction. The
	// UnlockHash is the hash of the UnlockConditions that must be fulfilled
	// in order to spend the output.
	SiacoinOutput struct {
		Value      Currency   `json:"value"`
		UnlockHash UnlockHash `json:"unlockhash"`
	}

	// A SiafundInput consumes a SiafundOutput and adds the siafunds to the set of
	// siafunds that can be spent in the transaction. The ParentID points to the
	// output that is getting consumed, and the UnlockConditions contain the rules
	// for spending the output. The UnlockConditions must match the UnlockHash of
	// the output.
	SiafundInput struct {
		ParentID         SiafundOutputID  `json:"parentid"`
		UnlockConditions UnlockConditions `json:"unlockconditions"`
		ClaimUnlockHash  UnlockHash       `json:"claimunlockhash"`
	}

	// A SiafundOutput holds a volume of siafunds. Outputs must be spent
	// atomically; that is, they must all be spent in the same transaction. The
	// UnlockHash is the hash of a set of UnlockConditions that must be fulfilled
	// in order to spend the output.
	//
	// When the SiafundOutput is spent, a SiacoinOutput is created, where:
	//
	//     SiacoinOutput.Value := (SiafundPool - ClaimStart) / 10,000
	//     SiacoinOutput.UnlockHash := SiafundOutput.ClaimUnlockHash
	//
	// When a SiafundOutput is put into a transaction, the ClaimStart must always
	// equal zero. While the transaction is being processed, the ClaimStart is set
	// to the value of the SiafundPool.
	SiafundOutput struct {
		Value      Currency   `json:"value"`
		UnlockHash UnlockHash `json:"unlockhash"`
		ClaimStart Currency   `json:"claimstart"`
	}

	// An UnlockHash is a specially constructed hash of the UnlockConditions type.
	// "Locked" values can be unlocked by providing the UnlockConditions that hash
	// to a given UnlockHash. See UnlockConditions.UnlockHash for details on how the
	// UnlockHash is constructed.
	UnlockHash crypto.Hash
)

func (t Transaction) marshalSiaNoSignatures(w io.Writer) {
	enc := encoding.NewEncoder(w)

	for i := range t.SiacoinInputs {
		t.SiacoinInputs[i].MarshalSia(w)
	}
	encoding.WriteInt(w, len((t.SiacoinOutputs)))
	for i := range t.SiacoinOutputs {
		t.SiacoinOutputs[i].MarshalSia(w)
	}
	encoding.WriteInt(w, len((t.FileContracts)))
	for i := range t.FileContracts {
		enc.Encode(t.FileContracts[i])
	}
	encoding.WriteInt(w, len((t.FileContractRevisions)))
	for i := range t.FileContractRevisions {
		enc.Encode(t.FileContractRevisions[i])
	}
	encoding.WriteInt(w, len((t.StorageProofs)))
	for i := range t.StorageProofs {
		enc.Encode(t.StorageProofs[i])
	}
	encoding.WriteInt(w, len((t.SiafundInputs)))
	for i := range t.SiafundInputs {
		enc.Encode(t.SiafundInputs[i])
	}
	encoding.WriteInt(w, len((t.SiafundOutputs)))
	for i := range t.SiafundOutputs {
		t.SiafundOutputs[i].MarshalSia(w)
	}
	encoding.WriteInt(w, len((t.MinerFees)))
	for i := range t.MinerFees {
		t.MinerFees[i].MarshalSia(w)
	}
	encoding.WriteInt(w, len((t.ArbitraryData)))
	for i := range t.ArbitraryData {
		encoding.WritePrefix(w, t.ArbitraryData[i])
	}
}

// ID returns the id of a transaction, which is taken by marshalling all of the
// fields except for the signatures and taking the hash of the result.
func (t Transaction) ID() TransactionID {
	var txid TransactionID
	h := crypto.NewHash()

	// Encode non-signature fields into hash.
	t.marshalSiaNoSignatures(h)

	// Encode signatures.
	encoding.WriteInt(h, len((t.TransactionSignatures)))
	for i := range t.TransactionSignatures {
		t.TransactionSignatures[i].MarshalSia(h)
	}

	h.Sum(txid[:0])
	return txid
}

// SiacoinOutputID returns the ID of a siacoin output at the given index,
// which is calculated by hashing the concatenation of the SiacoinOutput
// Specifier, all of the fields in the transaction (except the signatures),
// and output index.
func (t Transaction) SiacoinOutputID(i uint64) SiacoinOutputID {
	var id SiacoinOutputID
	h := crypto.NewHash()

	h.Write(SpecifierSiacoinOutput[:])
	t.marshalSiaNoSignatures(h) // Encode non-signature fields into hash.
	encoding.WriteUint64(h, i)  // Writes index of this output.

	h.Sum(id[:0])
	return id
}

// FileContractID returns the ID of a file contract at the given index, which
// is calculated by hashing the concatenation of the FileContract Specifier,
// all of the fields in the transaction (except the signatures), and the
// contract index.
func (t Transaction) FileContractID(i uint64) FileContractID {
	var id FileContractID
	h := crypto.NewHash()
	h.Write(SpecifierFileContract[:])
	t.marshalSiaNoSignatures(h) // Encode non-signature fields into hash.
	encoding.WriteUint64(h, i)  // Writes index of this output.

	h.Sum(id[:0])
	return id
}

// SiafundOutputID returns the ID of a SiafundOutput at the given index, which
// is calculated by hashing the concatenation of the SiafundOutput Specifier,
// all of the fields in the transaction (except the signatures), and output
// index.
func (t Transaction) SiafundOutputID(i uint64) SiafundOutputID {
	var id SiafundOutputID
	h := crypto.NewHash()
	h.Write(SpecifierSiafundOutput[:])
	t.marshalSiaNoSignatures(h) // Encode non-signature fields into hash.
	encoding.WriteUint64(h, i)  // Writes index of this output.

	h.Sum(id[:0])
	return id
}

// SiacoinOutputSum returns the sum of all the siacoin outputs in the
// transaction, which must match the sum of all the siacoin inputs. Siacoin
// outputs created by storage proofs and siafund outputs are not considered, as
// they were considered when the contract responsible for funding them was
// created.
func (t Transaction) SiacoinOutputSum() (sum Currency) {
	// Add the siacoin outputs.
	for _, sco := range t.SiacoinOutputs {
		sum = sum.Add(sco.Value)
	}

	// Add the file contract payouts.
	for _, fc := range t.FileContracts {
		sum = sum.Add(fc.Payout)
	}

	// Add the miner fees.
	for _, fee := range t.MinerFees {
		sum = sum.Add(fee)
	}

	return
}

// SiaClaimOutputID returns the ID of the SiacoinOutput that is created when
// the siafund output is spent. The ID is the hash the SiafundOutputID.
func (id SiafundOutputID) SiaClaimOutputID() SiacoinOutputID {
	return SiacoinOutputID(crypto.HashObject(id))
}
