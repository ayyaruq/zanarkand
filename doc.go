/*
Package zanarkand provides TCP packet reassembly for FFXIV network streams and some simple interfaces to access them.
The main use is for capturing FFXIV IPC Messages for parsing or analysis of game events. In FFXIV, communication
between the client and server is done via RPC. Each IPC message is identified by a Segment ID to identify it's payload
type, and some Segment-specific fields such as opcodes to identify IPC types.. One or more Messages are then wrapped
into a Frame, optionally compressed with ZLIB, and transmitted over TCP.
*/
package zanarkand
