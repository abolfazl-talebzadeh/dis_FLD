package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	fd "github.com/abolfazl-talebzadeh/goMods/fd"
	ld "github.com/abolfazl-talebzadeh/goMods/ld"
	disFLD "github.com/abolfazl-talebzadeh/goMods/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type NodeList struct {
	port    string
	IP      string
	conn    *grpc.ClientConn
	NotDead bool
}

type myNetwork struct {
	myNodeID   string
	myIP       string
	nodeList   map[int]NodeList
	leader     string
	alive      bool
	evtLeader  ld.MonLeaderDetector
	evtFailure fd.EvtFailureDetector
}

func NewmyNetwork(selfNode string, selfIp string, hbOut chan<- fd.Heartbeat) *myNetwork {
	nID := []int{}
	delta := time.Second
	nodeIndex, _ := strconv.Atoi(selfNode)
	nID = append(nID, nodeIndex)
	eLeader := ld.NewMonLeaderDetector(nID)
	eFailure := fd.NewEvtFailureDetector(nodeIndex, nID, eLeader, delta, hbOut)
	nodes := make(map[int]NodeList)
	nodeSample := NodeList{port: selfNode, IP: selfIp, conn: nil, NotDead: true}
	nodes[nodeIndex] = nodeSample
	return &myNetwork{myNodeID: selfNode, nodeList: nodes, leader: "", alive: true,
		evtLeader: *eLeader, evtFailure: *eFailure}
}
func main() {
	fmt.Println("Enter the port that you wish to listen on: ")
	hbOut := make(chan fd.Heartbeat, 1000)
	port := bufio.NewScanner(os.Stdin)
	port.Scan()
	hostname, _ := os.Hostname()
	myNet := NewmyNetwork(port.Text(), hostname, hbOut)
	id, _ := strconv.Atoi(port.Text())
	myNet.evtFailure.SetID(id)
	endpoint := hostname + ":" + port.Text()
	println(endpoint)
	go myNet.listenTCP(endpoint)
	myNet.evtFailure.Start()
	go func(m *myNetwork) {
		for {
			if !m.alive {
				continue
			}
			select {
			case hb := <-hbOut:
				if hb.Request {
					conn := m.nodeList[hb.To].conn
					c := disFLD.NewDistributedNetworkClient(conn)
					hbReq := fd2dsiFLD(hb)
					hbResp, err := c.HBExchange(context.Background(), &hbReq)
					if err == nil {
						m.evtFailure.DeliverHeartbeat(disFLD2fd(*hbResp))
					} else {
						continue
					}
				}
			default:
				continue
			}
		}
	}(myNet)
	for {
		theMenue(myNet)
	}
}

func (m *myNetwork) listenTCP(endpoint string) {
	lis, err := net.Listen("tcp", endpoint)
	conString := lis.Addr()
	if err != nil {
		log.Fatal("Listenning on ", endpoint, " wasn't successful!")
	} else {
		fmt.Println("started listening on ", conString)
	}
	ipport := strings.Split(conString.String(), ":")
	ip := ipport[0]
	//port,_ := strconv.Atoi(ipport[1])
	m.myIP = ip
	grpcServer := grpc.NewServer()
	disFLD.RegisterDistributedNetworkServer(grpcServer, m)
	err = grpcServer.Serve(lis)
	if err != nil {
		log.Fatal("couldn't strat the grpc server")
	}
	defer grpcServer.Stop()
}

func theMenue(m *myNetwork) {
	input := bufio.NewScanner(os.Stdin)
	fmt.Println("******************")
	fmt.Println("Enter your choice")
	fmt.Println("1. to connect")
	fmt.Println("1. to disconnect")
	fmt.Println("3. for the leader")
	fmt.Println("4. for the leader")
	fmt.Println("******************")
	fmt.Print("-> ")
	input.Scan()
	switch input.Text() {
	case "1":
		fmt.Println("======================")
		fmt.Println("You chose to connect")
		fmt.Println("======================")
		fmt.Print("Enter the port you want to connect to: ")
		input.Scan()
		m.NewClient(input.Text())
		ipport := strings.Split(input.Text(), ":")
		port := ipport[1]
		ip := ipport[0]
		selfNode := &disFLD.Node{NodeID: m.myNodeID, Port: m.myNodeID, IP: ip}
		outerPort, _ := strconv.Atoi(port)
		conn := m.nodeList[outerPort].conn
		c := disFLD.NewDistributedNetworkClient(conn)
		nodes, _ := c.NodeListExchange(context.Background(), selfNode)
		fmt.Printf("Nodes received from %s\n", input.Text())
		//connecting to nodes of the neighbouring node
		for _, i := range nodes.NodeID {
			for _, nodes := range m.nodeList {
				if nodes.port == i.Port {
					break
				} else {
					m.NewClient(i.IP + ":" + i.Port)
				}
			}
			fmt.Println("NodeID: ", i.NodeID)
			fmt.Println("Node Port: ", i.Port)
			fmt.Println("Node IP: ", i.IP)
		}
	case "2":
		fmt.Println("======================")
		fmt.Println("You chose to disconnect")
		fmt.Println("======================")
		m.alive = false
		m.evtLeader.Deactivate()
	case "3":
		fmt.Println("============================")
		fmt.Println("You chose to see the leader")
		fmt.Println("============================")
		fmt.Println("failure alive nodes: ", m.evtFailure.ShowAlive())
		fmt.Println("failure suspended nodes: ", m.evtFailure.ShowSuspended())
		fmt.Println("=============================================")
		fmt.Println("leader Nodes nodes: ", m.evtLeader.NodeID)
		fmt.Println("leader suspended nodes: ", m.evtLeader.ShowSuspected())
		fmt.Println("=============================================")
		fmt.Println("The current leader is: ", m.evtLeader.Leader())
		fmt.Println("=============================================")

	case "4":
		fmt.Println("====================================")
		fmt.Println("You chose to see the connected nodes")
		fmt.Println("====================================")
		for _, i := range m.nodeList {
			fmt.Println(i.port, "----", i.IP)
		}
	}
}

func (m *myNetwork) NewClient(endpoint string) {
	var conn *grpc.ClientConn
	var nodeIndex int
	//selfNode, _ := strconv.Atoi(m.myNodeID)
	conn, err := grpc.Dial(endpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal("couldn't connect to server: ", err)
	}
	ipport := strings.Split(endpoint, ":")
	ip := ipport[0]
	port := ipport[1]
	nodeIndex, _ = strconv.Atoi(port)
	sample := &NodeList{port: port, IP: ip, conn: conn, NotDead: true}
	//fmt.Println(sample)
	m.nodeList[nodeIndex] = *sample
	//println("before adding node --> ", nodeIndex)
	// for index, alive := range m.evtFailure.ShowAlive() {
	// 	println("Alive: ", index, "   ", alive)
	// }
	// for index, sus := range m.evtFailure.ShowSuspended() {
	// 	println("suspended: ", index, "    ", sus)
	// }
	m.evtFailure.AddNewNode(nodeIndex)
	m.evtLeader.AddNewNode(nodeIndex)
	m.evtFailure.UpdateSR(&m.evtLeader)
	//println("after adding node --> ", nodeIndex)
	// for index, alive := range m.evtFailure.ShowAlive() {
	// 	println("Alive: ", index, "   ", alive)
	// }
	// for index, sus := range m.evtFailure.ShowSuspended() {
	// 	println("suspended: ", index, "    ", sus)
	// }
}

func (m *myNetwork) NodeListExchange(ctx context.Context, Node *disFLD.Node) (*disFLD.NodeList, error) {
	for _, nodes := range m.nodeList {
		if nodes.port == Node.Port {
			break
		} else {
			m.NewClient(Node.IP + ":" + Node.Port)
		}
	}
	nodeList := new(disFLD.NodeList)
	for _, i := range m.nodeList {
		nodeList.NodeID = append(nodeList.NodeID, &disFLD.Node{NodeID: i.port, Port: i.port, IP: i.IP})
	}
	return nodeList, nil
}

func (m *myNetwork) HBExchange(ctx context.Context, hb *disFLD.HeartBeat) (*disFLD.HeartBeat, error) {
	if m.alive {
		//println("HBExchange-->", " from: ", hb.From, " to: ", hb.To, " resuest: ", hb.Request)
		return &disFLD.HeartBeat{From: hb.To, To: hb.From, Request: false}, nil
	} else {
		return nil, errors.New("sorry I'm dead")
	}
}

func disFLD2fd(hb disFLD.HeartBeat) fd.Heartbeat {
	from, _ := strconv.Atoi(hb.From)
	_To, _ := strconv.Atoi(hb.To)
	return fd.Heartbeat{From: from, To: _To, Request: hb.Request}
}

func fd2dsiFLD(hb fd.Heartbeat) disFLD.HeartBeat {
	from := strconv.Itoa(hb.From)
	_To := strconv.Itoa(hb.To)
	return disFLD.HeartBeat{
		From:                 from,
		To:                   _To,
		Request:              hb.Request,
		XXX_NoUnkeyedLiteral: struct{}{},
		XXX_unrecognized:     []byte{},
		XXX_sizecache:        0,
	}
}
