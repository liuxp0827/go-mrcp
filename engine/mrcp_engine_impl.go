package engine

import (
	"github.com/navi-tt/go-mrcp/mpf"
	"github.com/navi-tt/go-mrcp/mrcp"
	"github.com/navi-tt/go-mrcp/mrcp/message"
)

/** Create engine */
func MRCPEngineCreate(rid mrcp.MRCPResourceId, obj interface{}, vtable *MRCPEngineMethodVTable) *MRCPEngine {
	return &MRCPEngine{}
}

/** Send engine open response */
func (engine *MRCPEngine) MRCPEngineOpenRespond(status bool) error {
	return engine.EventVTable.OnOpen(engine, status)
}

/** Send engine close response */
func (engine *MRCPEngine) MRCPEngineCloseRespond() error {
	return engine.EventVTable.OnClose(engine)
}

/** Get engine config */
func (engine *MRCPEngine) MRCPEngineConfigGet() *MRCPEngineConfig {
	return engine.Config
}

/** Get engine param by name */
func (engine *MRCPEngine) MRCPEngineParamGet(name string) string {
	return ""
}

/** Create engine channel */
func (engine *MRCPEngine) MRCPEngineChannelCreate(methodVTable *MRCPEngineMethodVTable, methodObj interface{}, termination *mpf.Termination) *MRCPEngineChannel {
	return nil
}

/** Create audio termination */
func MRCPEngineAudioTerminationCreate(obj interface{}, streamVTable *mpf.AudioStreamVTable, capabilities *mpf.StreamCapabilities) *mpf.Termination {
	return nil
}

/** Create engine channel and source media termination
 * @deprecated @see mrcp_engine_channel_create() and mrcp_engine_audio_termination_create()
 */
func (engine *MRCPEngine) MRCPEngineSourceChannelCreate(channelVTable *MRCPEngineChannelMethodVTable, streamVTable *mpf.AudioStreamVTable,
	methodObj interface{}, codecDescriptor *mpf.CodecDescriptor) *MRCPEngineChannel {
	return nil
}

/** Create engine channel and sink media termination
 * @deprecated @see mrcp_engine_channel_create() and mrcp_engine_audio_termination_create()
 */
func (engine *MRCPEngine) MRCPEngineSinkChannelCreate(channelVTable *MRCPEngineChannelMethodVTable, streamVTable *mpf.AudioStreamVTable,
	methodObj interface{}, codecDescriptor *mpf.CodecDescriptor) *MRCPEngineChannel {
	return nil
}

/** Send channel open response */
func (channel *MRCPEngineChannel) MRCPEngineChannelOpenRespond(status bool) error {
	return channel.EventVTable.OnOpen(channel, status)
}

/** Send channel close response */
func (channel *MRCPEngineChannel) MRCPEngineChannelCloseRespond() error {
	return channel.EventVTable.OnClose(channel)
}

/** Send response/event message */
func (channel *MRCPEngineChannel) MRCPEngineChannelMessageSend(message *message.MRCPMessage) error {
	return channel.EventVTable.OnMessage(channel, message)
}

/** Get channel identifier */
func (channel *MRCPEngineChannel) MRCPEngineChannelIdGet() string {
	return channel.Id
}

/** Get MRCP version channel is created in the scope of */
func (channel *MRCPEngineChannel) MRCPEngineChannelVersionGet() mrcp.Version {
	return channel.Version
}

/** Get codec descriptor of the audio source stream */
func (channel *MRCPEngineChannel) MRCPEngineSourceStreamCodecGet() *mpf.CodecDescriptor {
	return nil
}

/** Get codec descriptor of the audio sink stream */
func (channel *MRCPEngineChannel) MRCPEngineSinkStreamCodecGet() *mpf.CodecDescriptor {
	return nil
}
