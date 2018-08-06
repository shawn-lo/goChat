package main

import (
	"fmt"
	"time"
)

// 5 min
var VALID_DURATION time.Duration = 5*1000*1000*1000*60
// Hub maintains the set of active clients and broadcasts messages to the
// clients.
type Hub struct {
	// Registered customers. HashKey(Client)
	customers map[string]*Customer

	// Registered agents. HashKey(Agent)
	agents map[string]*Agent

	// Register requests from the customers.
	cRegister chan *Customer

	// Register requests from the agents.
	aRegister chan *Agent

	// Unregister requests from customers.
	cUnregister chan *Customer

	// Unregister requests from agnents.
	aUnregister chan *Agent

	cusMapToAgent map[*Customer]*Agent

	agentMapToCus map[*Agent]*Customer

	availableAgents []*Agent

	waitingCustomers int8

	publicChan chan []byte
}

func newHub() *Hub {
	return &Hub{
		cRegister:   make(chan *Customer),
		aRegister:   make(chan *Agent),
		cUnregister: make(chan *Customer),
		aUnregister: make(chan *Agent),
		customers:   make(map[string]*Customer),
		agents:      make(map[string]*Agent),
		availableAgents: make([]*Agent, 0),
		cusMapToAgent: make(map[*Customer]*Agent),
		agentMapToCus: make(map[*Agent]*Customer),
		waitingCustomers: 0,
		publicChan: make(chan []byte, 5),
	}
}

func (h *Hub) nextAvailableAgent() *Agent{
	if len(h.availableAgents) > 0 {
		a := h.availableAgents[0]
		h.availableAgents = h.availableAgents[1:]
		return a
	}
	return nil
}

func (h *Hub) run() {
	for {
		select {
		case customer := <-h.cRegister:
			h.customers[customer.HashKey()] = customer
			// give notification
			if len(h.availableAgents) > 0 {
				fmt.Println(len(h.availableAgents))
				a := h.availableAgents[0]
				h.availableAgents = h.availableAgents[1:]
				fmt.Println(len(h.availableAgents))
				h.agentMapToCus[a] = customer
				h.cusMapToAgent[customer] = a
			} else {
				//h.waitingCustomers++
				h.publicChan<- []byte("A new client login.")
			}
		case agent := <-h.aRegister:
			h.agents[agent.HashKey()] = agent
			h.availableAgents = append(h.availableAgents, agent)
		case customer := <-h.cUnregister:
			fmt.Println("Customer Closed")
			cusKey := customer.HashKey()
			if _, ok := h.customers[cusKey]; ok {
				if a, ok := h.cusMapToAgent[customer]; ok {
					if a != nil {
						a.conn.Close()
						delete(h.cusMapToAgent, customer)
					} else {
						fmt.Println("This customer leaves before match.")
					}
				}
				delete(h.customers, cusKey)
				close(customer.recv)
			}
		case agent := <-h.aUnregister:
			fmt.Println("Agent Closed")
			agentKey := agent.HashKey()
			if _, ok := h.agents[agentKey]; ok {
				if c, ok := h.agentMapToCus[agent]; ok {
					if c != nil {
						c.conn.Close()
						delete(h.agentMapToCus, agent)
					} else {
						fmt.Println("This agent leaves before a customer coming.")
					}
				}
				delete(h.agents, agentKey)
				close(agent.recv)
			}
		case message := <-h.publicChan:
			fmt.Printf("There are %d agents\n", len(h.agents))
			for agent := range h.agents {
				select {
				case h.agents[agent].recv <- message:
				}
			}
		}

	}
}
