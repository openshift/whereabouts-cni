package allocate

import (
	"fmt"
	"math"
	"net"

	"github.com/k8snetworkplumbingwg/whereabouts/pkg/logging"
	"github.com/k8snetworkplumbingwg/whereabouts/pkg/types"
)

// AssignmentError defines an IP assignment error.
type AssignmentError struct {
	firstIP       net.IP
	lastIP        net.IP
	ipnet         net.IPNet
	excludeRanges []string
}

func (a AssignmentError) Error() string {
	return fmt.Sprintf("Could not allocate IP in range: ip: %v / - %v / range: %s / excludeRanges: %v",
		a.firstIP, a.lastIP, a.ipnet.String(), a.excludeRanges)
}

// AssignIP assigns an IP using a range and a reserve list.
func AssignIP(ipamConf types.RangeConfiguration, reservelist []types.IPReservation, containerID string, podRef string) (net.IPNet, []types.IPReservation, error) {

	// Setup the basics here.
	_, ipnet, _ := net.ParseCIDR(ipamConf.Range)

	newip, updatedreservelist, err := IterateForAssignment(*ipnet, ipamConf.RangeStart, ipamConf.RangeEnd, reservelist, ipamConf.OmitRanges, containerID, podRef)
	if err != nil {
		return net.IPNet{}, nil, err
	}

	return net.IPNet{IP: newip, Mask: ipnet.Mask}, updatedreservelist, nil
}

// DeallocateIP removes allocation from reserve list. Returns the updated reserve list and the deallocated IP.
func DeallocateIP(reservelist []types.IPReservation, containerID string) ([]types.IPReservation, net.IP) {
	index := getMatchingIPReservationIndex(reservelist, containerID)
	if index < 0 {
		// Allocation not found. Return the original reserve list and nil IP.
		return reservelist, nil
	}

	ip := reservelist[index].IP
	logging.Debugf("Deallocating given previously used IP: %v", ip.String())

	return removeIdxFromSlice(reservelist, index), ip
}

func getMatchingIPReservationIndex(reservelist []types.IPReservation, id string) int {
	for idx, v := range reservelist {
		if v.ContainerID == id {
			return idx
		}
	}
	return -1
}

func removeIdxFromSlice(s []types.IPReservation, i int) []types.IPReservation {
	s[i] = s[len(s)-1]
	return s[:len(s)-1]
}

// byteSliceAdd adds ar1 to ar2
// note: ar1/ar2 should be 16-length array
func byteSliceAdd(ar1, ar2 []byte) ([]byte, error) {
	if len(ar1) != len(ar2) {
		return nil, fmt.Errorf("byteSliceAdd: bytes array mismatch: %v != %v", len(ar1), len(ar2))
	}
	carry := uint(0)

	sumByte := make([]byte, 16)
	for n := range ar1 {
		sum := uint(ar1[15-n]) + uint(ar2[15-n]) + carry
		carry = 0
		if sum > 255 {
			carry = 1
		}
		sumByte[15-n] = uint8(sum)
	}

	return sumByte, nil
}

// byteSliceSub subtracts ar2 from ar1. This function assumes that ar1 > ar2
// note: ar1/ar2 should be 16-length array
func byteSliceSub(ar1, ar2 []byte) ([]byte, error) {
	if len(ar1) != len(ar2) {
		return nil, fmt.Errorf("byteSliceSub: bytes array mismatch")
	}
	carry := int(0)

	sumByte := make([]byte, 16)
	for n := range ar1 {
		var sum int
		sum = int(ar1[15-n]) - int(ar2[15-n]) - carry
		if sum < 0 {
			sum = 0x100 - int(ar1[15-n]) - int(ar2[15-n]) - carry
			carry = 1
		} else {
			carry = 0
		}
		sumByte[15-n] = uint8(sum)
	}

	return sumByte, nil
}

func ipAddrToUint64(ip net.IP) uint64 {
	num := uint64(0)
	ipArray := []byte(ip)
	for n := range ipArray {
		num = num << 8
		num = uint64(ipArray[n]) + num
	}
	return num
}

func ipAddrFromUint64(num uint64) net.IP {
	idxByte := make([]byte, 16)
	i := num
	for n := range idxByte {
		idxByte[15-n] = byte(0xff & i)
		i = i >> 8
	}
	return net.IP(idxByte)
}

// IPGetOffset gets offset between ip1 and ip2. This assumes ip1 > ip2 (from IP representation point of view)
func IPGetOffset(ip1, ip2 net.IP) uint64 {
	if ip1.To4() == nil && ip2.To4() != nil {
		return 0
	}

	if ip1.To4() != nil && ip2.To4() == nil {
		return 0
	}

	if len([]byte(ip1)) != len([]byte(ip2)) {
		return 0
	}

	ipOffset, _ := byteSliceSub([]byte(ip1.To16()), []byte(ip2.To16()))
	return ipAddrToUint64(ipOffset)
}

// IPAddOffset show IP address plus given offset
func IPAddOffset(ip net.IP, offset uint64) net.IP {
	// Check IPv4 and its offset range
	if ip.To4() != nil && offset >= math.MaxUint32 {
		return nil
	}

	// make pseudo IP variable for offset
	idxIP := ipAddrFromUint64(offset)

	b, _ := byteSliceAdd([]byte(ip.To16()), []byte(idxIP))
	return net.IP(b)
}

// IterateForAssignment iterates given an IP/IPNet and a list of reserved IPs
func IterateForAssignment(ipnet net.IPNet, rangeStart net.IP, rangeEnd net.IP, reserveList []types.IPReservation, excludeRanges []string, containerID string, podRef string) (net.IP, []types.IPReservation, error) {
	firstIP := rangeStart.To16()
	var lastIP net.IP
	if rangeEnd != nil {
		lastIP = rangeEnd.To16()
	} else {
		var err error
		firstIP, lastIP, err = GetIPRange(rangeStart, ipnet)
		if err != nil {
			logging.Errorf("GetIPRange request failed with: %v", err)
			return net.IP{}, reserveList, err
		}
	}
	logging.Debugf("IterateForAssignment input >> ip: %v | ipnet: %v | first IP: %v | last IP: %v", rangeStart, ipnet, firstIP, lastIP)

	reserved := make(map[string]bool)
	for _, r := range reserveList {
		reserved[r.IP.String()] = true
	}

	// Build excluded list, "192.168.2.229/30", "192.168.1.229/30".
	excluded := []*net.IPNet{}
	for _, v := range excludeRanges {
		subnet, err := parseExcludedRange(v)
		if err != nil {
			return net.IP{}, reserveList, fmt.Errorf("could not parse exclude range, err: %q", err)
		}
		excluded = append(excluded, subnet)
	}

	// Iterate every IP address in the range
	var assignedip net.IP
	performedassignment := false
	endip := IPAddOffset(lastIP, uint64(1))
	for i := firstIP; ipnet.Contains(i) && !i.Equal(endip); i = IPAddOffset(i, uint64(1)) {
		// if already reserved, skip it
		if reserved[i.String()] {
			continue
		}

		// Lastly, we need to check if this IP is within the range of excluded subnets
		isAddrExcluded := false
		for _, subnet := range excluded {
			if subnet.Contains(i) {
				isAddrExcluded = true
				firstExcluded, _, _ := net.ParseCIDR(subnet.String())
				_, lastExcluded, _ := GetIPRange(firstExcluded, *subnet)
				if lastExcluded != nil {
					if i.To4() != nil {
						// exclude broadcast address
						i = IPAddOffset(lastExcluded, uint64(1))
					} else {
						i = lastExcluded
					}
					logging.Debugf("excluding %v and moving to the next available ip: %v", subnet, i)
				}
			}
		}
		if isAddrExcluded {
			continue
		}

		// Ok, this one looks like we can assign it!
		performedassignment = true

		assignedip = i
		logging.Debugf("Reserving IP: |%v|", assignedip.String()+" "+containerID)
		reserveList = append(reserveList, types.IPReservation{IP: assignedip, ContainerID: containerID, PodRef: podRef})
		break
	}

	if !performedassignment {
		return net.IP{}, reserveList, AssignmentError{firstIP, lastIP, ipnet, excludeRanges}
	}

	return assignedip, reserveList, nil
}

func mergeIPAddress(net, host []byte) ([]byte, error) {
	if len(net) != len(host) {
		return nil, fmt.Errorf("not matched")
	}
	addr := append([]byte{}, net...)
	for i := range net {
		addr[i] = net[i] | host[i]
	}
	return addr, nil
}

// GetIPRange returns the first and last IP in a range
func GetIPRange(ip net.IP, ipnet net.IPNet) (net.IP, net.IP, error) {
	mask := ipnet.Mask
	ones, bits := mask.Size()
	masklen := bits - ones

	// Error when the mask isn't large enough.
	if masklen < 2 {
		return nil, nil, fmt.Errorf("net mask is too short, must be 2 or more: %v", masklen)
	}

	// get network part
	network := ip.Mask(ipnet.Mask)
	// get bitmask for host
	hostMask := net.IPMask(append([]byte{}, ipnet.Mask...))
	for i, n := range hostMask {
		hostMask[i] = ^n
	}
	// get host part of ip
	first := ip.Mask(net.IPMask(hostMask))
	// if ip is just same as ipnet.IP, i.e. just network address,
	// increment it for start ip
	if ip.Equal(ipnet.IP) {
		first[len(first)-1] = 0x1
	}
	// calculate last byte
	last := hostMask
	// if IPv4 case, decrement 1 for broadcasting address
	if ip.To4() != nil {
		last[len(last)-1]--
	}
	// get first ip and last ip based on network part + host part
	firstIPbyte, _ := mergeIPAddress([]byte(network), first)
	lastIPbyte, _ := mergeIPAddress([]byte(network), last)
	firstIP := net.IP(firstIPbyte).To16()
	lastIP := net.IP(lastIPbyte).To16()

	return firstIP, lastIP, nil
}

// IsIPv4 checks if an IP is v4.
func IsIPv4(checkip net.IP) bool {
	return checkip.To4() != nil
}

// parseExcludedRange parses a provided string to a net.IPNet.
// If the provided string is a valid CIDR, return the net.IPNet for that CIDR.
// If the provided string is a valid IP address, add the /32 or /128 prefix to form the CIDR and return the net.IPNet.
// Otherwise, return the error.
func parseExcludedRange(s string) (*net.IPNet, error) {
	// Try parsing CIDRs.
	_, subnet, err := net.ParseCIDR(s)
	if err == nil {
		return subnet, nil
	}
	// The user might have given a single IP address, try parsing that - if it does not parse, return the error that
	// we got earlier.
	ip := net.ParseIP(s)
	if ip == nil {
		return nil, err
	}
	// If the address parses, check if it's IPv4 or IPv6 and add the correct prefix.
	if ip.To4() != nil {
		_, subnet, err = net.ParseCIDR(fmt.Sprintf("%s/32", s))
	} else {
		_, subnet, err = net.ParseCIDR(fmt.Sprintf("%s/128", s))
	}
	return subnet, err
}
