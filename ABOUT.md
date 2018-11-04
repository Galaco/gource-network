## Gource network client
A lot of the information obtained here is based on work performed by [http://github.com/Leystryku](Leystryku) here: []().


### Connection information
#### Packet types
There are 3 packet types, based on a 4byte int in the packet header. The flag can be:
* PacketHeaderFlagQuery (-1) - A connectionless packet. Used in initial connection steps
* PacketHeaderFlagSplit (-2) - Packet data spans multiple packet(s)
* PacketHeaderFlagCompressed (-3) - Packet data is compressed



### Connecting to a server
##### Initial connection
Creating a connection to the server takes a few back and forth steps. The objective of our client
is to prove our identity, and obtain the current server information.

The initial steps are as follows:

* 1: Naturally the client makes the first move. We send an initial packet to the server with our
challenge token. The packet body looks like this:
```
{
    -1           // long
    'q'          // byte
    0x0B5B1842   // challenge
    "0000000000" // string
}
```
* 2: Wait for packet with PacketHeaderFlagQuery. There are multiple different packet types we
could get at this point. Each type is identifiable by a flag in the head indicating the
state of the connection:
    * '9'. The connection has been refused.
    * 'A'. (`A2A_GETCHALLENGE`). Ideally this is what we want.

A packet with type 'A' should contain contents like this:
```
    magicNumber             // long
    serverChallenge         // long
    ourChallenge            // long
    authProtocol            // long
    encryptionKeySize       // short hope its 0
    steamkey_encryptionkey  // char of length encryptionKeySize
    serversteamid           // char of length 2048
    vacsecured              // byte
```

* 3: Now we can craft a response based on the contents of our previously received packet.
We will send a new packet with the following data structure:
```
    -1                  //magic long
    'k'                 //C2S_CONNECT
    0x18                //protocol ver
    0x03                //auth protocol 0x03 = PROTOCOL_STEAM, 0x02 = PROTOCOL_HASHEDCDKEY, 0x01=PROTOCOL_AUTHCERTIFICATE
    serverChallenge     //comes from previous response
    ourChallenge        //comes from previous response
    2729496039          //magic ubitlong
    nickname            //nickname 255 length
    password            //password 255 length
    "2000"              //game version

    242                 //magic number
    steamId             //long long (steamid64)
    #if steamAuthSessionTicket provides a key
        key             // key length provided in ticket info
    #endif
```

* 4: We will wait for another packet with header info PacketHeaderFlagQuery. This time, we
want packet type as 'B' (`S2C_CONNECTION`).
We don't seem to care about the actual contents of this packet. So we can respond with
the following contents as another packet to send
```
    6               //Unsigned long (only first 6 bits)
    2               //byte
    -1              //long
    4               //Unsigned long (only first 6 bits)
    "VModEnable 1"  //string
    4               //Unsigned long (only first 6 bits)
    "vban 0 0 0 0"  //string
```

After this, we can expect to receive packets from the server as a regular client.