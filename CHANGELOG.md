|Date      |Issue |Description                                                                                              |
|----------|------|---------------------------------------------------------------------------------------------------------|
|2019/01/08|      |Release 0.4.2                                                                                            |
|2019/01/08|78    |Improve robustness of event handling logic                                                               |
|2019/01/08|      |Release 0.4.1                                                                                            |
|2019/01/08|      |Update `go-choria` dependency to resolve panics during connection close                                  |
|2019/01/04|      |Release 0.4.0                                                                                            |
|2019/01/04|72    |Reject Choria client certificate patterns while signing nodes                                            |
|2018/12/27|70    |Improve splay related logging during restart                                                             |
|2018/12/27|      |Release 0.3.2                                                                                            |
|2018/12/26|67    |When restarting the Server in some cases the splay till restart could be 5k hours, it's now 2 seconds    |
|2018/12/14|      |Release 0.3.1                                                                                            |
|2018/12/10|59    |Retry `rpcutil#inventory` up to 5 times during deploy to report fewer node provision failures            |
|2018/12/07|58    |Use correct permissions when creating SSL files during provisioning - all were `0700`                    |
|2018/11/27|      |Release 0.3.0                                                                                            |
|2018/10/01|53    |Initial support for self-updating Choria                                                                 |
|2018/09/07|      |Release 0.2.2                                                                                            |
|2018/09/07|48    |Ensure the choria config is found on el7 nodes                                                           |
|2018/08/26|      |Release 0.2.1                                                                                            |
|2018/08/26|44    |Use lifecycle project to manage lifecycle events                                                         |
|2018/08/25|      |Release 0.2.0                                                                                            |
|2018/08/25|38    |Listen for node life cycle events where component is `provision_mode_server`                             |
|2018/08/24|36    |Emit a startup life cycle event with component `provisioner`                                             |
|2018/08/09|      |Release 0.1.0                                                                                            |
|2018/08/09|32    |Record successfully provisioned node counts                                                              |
|2018/08/09|24    |Record busy workers in stats                                                                             |
|2018/08/09|29    |Avoid nil pointer error when PKI feature is disabled                                                     |
|2018/08/09|28    |Increase max token length to 128                                                                         |
|2018/08/09|25    |Ensure helper run duration stat is registered                                                            |
|2018/08/09|      |Release 0.0.3                                                                                            |
|2018/08/09|21    |Fix various el6 startup issues                                                                           |
|2018/08/09|19    |Embed the DDL files in the binary to ease deployment                                                     |
|2018/08/09|17    |Create the pidfile late in the life cycle and delete it on exit                                          |
|2018/08/09|      |Release 0.0.2                                                                                            |
|2018/08/08|13    |Add a go-backplane instance                                                                              |
|2018/08/07|      |Release 0.0.1                                                                                            |
