package ics23

import (
	"bytes"
	"encoding/hex"
	"testing"
)

func TestLeafOp(t *testing.T) {
	cases := map[string]struct {
		op       *LeafOp
		key      []byte
		value    []byte
		isErr    bool
		expected []byte
	}{
		"hash foobar": {
			op: &LeafOp{
				Hash: HashOp_SHA256,
				// no prehash, no length prefix
			},
			key:   []byte("foo"),
			value: []byte("bar"),
			// echo -n foobar | sha256sum
			expected: fromHex("c3ab8ff13720e8ad9047dd39466b3c8974e592c2fa383d4a3960714caef0c4f2"),
		},
		"hash foobaz, sha-512": {
			op: &LeafOp{
				Hash: HashOp_SHA512,
				// no prehash, no length prefix
			},
			key:   []byte("foo"),
			value: []byte("baz"),
			// echo -n foobaz | sha512sum
			expected: fromHex("4f79f191298ec7461d60136c60f77c2ae8ddd85dbf6168bb925092d51bfb39b559219b39ae5385ba04946c87f64741385bef90578ea6fe6dac85dbf7ad3f79e1"),
		},
		"hash foobar (different break)": {
			op: &LeafOp{
				Hash: HashOp_SHA256,
				// no prehash, no length prefix
			},
			key:   []byte("f"),
			value: []byte("oobar"),
			// echo -n foobar | sha256sum
			expected: fromHex("c3ab8ff13720e8ad9047dd39466b3c8974e592c2fa383d4a3960714caef0c4f2"),
		},
		"requires key": {
			op: &LeafOp{
				Hash: HashOp_SHA256,
				// no prehash, no length prefix
			},
			key:   []byte("foo"),
			isErr: true,
		},
		"requires value": {
			op: &LeafOp{
				Hash: HashOp_SHA256,
				// no prehash, no length prefix
			},
			value: []byte("bar"),
			isErr: true,
		},
		"hash with length prefix": {
			op: &LeafOp{
				Hash:   HashOp_SHA256,
				Length: LengthOp_VAR_PROTO,
				// no prehash
			},
			// echo -n food | xxs -ps
			// and manually compute length byte
			key:   []byte("food"),             // 04666f6f64
			value: []byte("some longer text"), // 10736f6d65206c6f6e6765722074657874
			// echo -n 04666f6f6410736f6d65206c6f6e6765722074657874 | xxd -r -p | sha256sum
			expected: fromHex("b68f5d298e915ae1753dd333da1f9cf605411a5f2e12516be6758f365e6db265"),
		},
		"hash with prehash and length prefix": {
			op: &LeafOp{
				Hash:         HashOp_SHA256,
				Length:       LengthOp_VAR_PROTO,
				PrehashValue: HashOp_SHA256,
			},
			key: []byte("food"), // 04666f6f64
			// TODO: this is hash, then length....
			// echo -n yet another long string | sha256sum
			value: []byte("yet another long string"), // 20a48c2d4f67b9f80374938535285ed285819d8a5a8fc1fccd1e3244e437cf290d
			// echo -n 04666f6f6420a48c2d4f67b9f80374938535285ed285819d8a5a8fc1fccd1e3244e437cf290d | xxd -r -p | sha256sum
			expected: fromHex("87e0483e8fb624aef2e2f7b13f4166cda485baa8e39f437c83d74c94bedb148f"),
		},
		"hash with length prefix (fixed 32-bit little-endian encoding)": {
			op: &LeafOp{
				Hash:   HashOp_SHA256,
				Length: LengthOp_FIXED32_LITTLE,
				// no prehash
			},
			// echo -n food | xxs -ps
			// and manually compute length bytes
			key:   []byte("food"),             // 04000000666f6f64
			value: []byte("some longer text"), // 10000000736f6d65206c6f6e6765722074657874
			// echo -n 04000000666f6f6410000000736f6d65206c6f6e6765722074657874 | xxd -r -p | sha256sum
			expected: fromHex("c853652437be02501c674744bf2a2b45d92a0a9f29c4b1044010fb3e2d43a949"),
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			res, err := tc.op.Apply(tc.key, tc.value)
			// short-circuit with error case
			if tc.isErr {
				if err == nil {
					t.Fatal("Expected error, but got none")
				}
				return
			}

			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(res, tc.expected) {
				t.Errorf("Bad result: %s vs %s", toHex(res), toHex(tc.expected))
			}
		})
	}
}

func TestInnerOp(t *testing.T) {
	cases := map[string]struct {
		op       *InnerOp
		child    []byte
		isErr    bool
		expected []byte
	}{
		"requires child": {
			op: &InnerOp{
				Hash:   HashOp_SHA256,
				Prefix: fromHex("0123456789"),
				Suffix: fromHex("deadbeef"),
			},
			isErr: true,
		},
		"hash child with prefix and suffix": {
			op: &InnerOp{
				Hash:   HashOp_SHA256,
				Prefix: fromHex("0123456789"),
				Suffix: fromHex("deadbeef"),
			},
			child: fromHex("00cafe00"),
			// echo -n 012345678900cafe00deadbeef | xxd -r -p | sha256sum
			expected: fromHex("0339f76086684506a6d42a60da4b5a719febd4d96d8b8d85ae92849e3a849a5e"),
		},
		"hash child with only prefix": {
			op: &InnerOp{
				Hash:   HashOp_SHA256,
				Prefix: fromHex("00204080a0c0e0"),
			},
			child: fromHex("ffccbb997755331100"),
			// echo -n 00204080a0c0e0ffccbb997755331100 | xxd -r -p | sha256sum
			expected: fromHex("45bece1678cf2e9f4f2ae033e546fc35a2081b2415edcb13121a0e908dca1927"),
		},
		"hash child with only suffix": {
			op: &InnerOp{
				Hash:   HashOp_SHA256,
				Suffix: []byte(" just kidding!"),
			},
			child: []byte("this is a sha256 hash, really...."),
			// echo -n 'this is a sha256 hash, really.... just kidding!'  | sha256sum
			expected: fromHex("79ef671d27e42a53fba2201c1bbc529a099af578ee8a38df140795db0ae2184b"),
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			res, err := tc.op.Apply(tc.child)
			// short-circuit with error case
			if tc.isErr {
				if err == nil {
					t.Fatal("Expected error, but got none")
				}
				return
			}

			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(res, tc.expected) {
				t.Errorf("Bad result: %s vs %s", toHex(res), toHex(tc.expected))
			}
		})
	}
}

func TestDoHash(t *testing.T) {
	cases := map[string]struct {
		hashOp       HashOp
		preimage     string
		expectedHash string
	}{
		"sha256": {
			hashOp:   HashOp_SHA256,
			preimage: "food",
			// echo -n food | sha256sum
			expectedHash: "c1f026582fe6e8cb620d0c85a72fe421ddded756662a8ec00ed4c297ad10676b",
		},
		"ripemd160": {
			hashOp:   HashOp_RIPEMD160,
			preimage: "food",
			// echo -n food | openssl dgst -rmd160 -hex | cut -d' ' -f2
			expectedHash: "b1ab9988c7c7c5ec4b2b291adfeeee10e77cdd46",
		},
		"bitcoin": {
			hashOp:   HashOp_BITCOIN,
			preimage: "food",
			// echo -n c1f026582fe6e8cb620d0c85a72fe421ddded756662a8ec00ed4c297ad10676b | xxd -r -p | openssl dgst -rmd160 -hex
			expectedHash: "0bcb587dfb4fc10b36d57f2bba1878f139b75d24",
		},
		"sha512_256": {
			hashOp:   HashOp_SHA512_256,
			preimage: "food",
			// echo -n food | openssl dgst -sha512-256 -hex | cut -d' ' -f2
			expectedHash: "5b3a452a6acbf1fc1e553a40c501585d5bd3cca176d562e0a0e19a3c43804e88",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			res, err := DoHash(tc.hashOp, []byte(tc.preimage))
			if err != nil {
				t.Fatal(err)
			}
			hexRes := hex.EncodeToString(res)
			if hexRes != tc.expectedHash {
				t.Fatalf("Expected %s got %s", tc.expectedHash, hexRes)
			}
		})
	}
}

func fromHex(data string) []byte {
	res, err := hex.DecodeString(data)
	if err != nil {
		panic(err)
	}
	return res
}

func toHex(data []byte) string {
	return hex.EncodeToString(data)
}
