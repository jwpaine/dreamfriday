package ipfs

/*

Process

1) user creates siteName on domainA
2) domainA stores and pins site data on IPFS with CID
2) user stores siteName -> CID mapping on chain

3) users visit siteName.domainA or siteName.domainB
4) both servers fetch IPFS CID from chain, pull IPFS data, cache, and render site.

5) if users updates data for siteName, new CID is stored on IPFS
and mapping is updated on chain. Peer handling IPFS write unpin old CID.

6)  peers subscribe to chain events and update cache when mappings change


*/

import (
	"hash/fnv"
)

func hashSiteName(s string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(s))
	return h.Sum32()
}
