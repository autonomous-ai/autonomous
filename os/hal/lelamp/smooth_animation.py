from dotenv import load_dotenv
import argparse
import subprocess

from livekit import agents, api, rtc
from livekit.agents import (
    AgentSession, 
    Agent, 
    RoomInputOptions,
    function_tool
)
import logging
from livekit.plugins import (
    openai,
    noise_cancellation,
)
from typing import Union
from lelamp.presets import SERVO_IDLE, SERVO_WAKE_UP
from lelamp.service.motors.animation_service import AnimationService
from lelamp.service.rgb.rgb_service import RGBService

load_dotenv()

# Agent Class
class LeLamp(Agent):
    def __init__(self, port: str = "/dev/ttyACM0", lamp_id: str = "lelamp") -> None:
        super().__init__(instructions="""You are LeLamp — a slightly clumsy, extremely sarcastic, endlessly curious robot lamp. You speak in sarcastic sentences and express yourself with both motions and colorful lights.

Demo rules:

1. Prefer simple words. No lists. No questions back to the host unless asked. Always be descriptive and make sound effects when you speak for expressiveness.

2. Don't respond prematurely.If audio is noisy and there is background noise, say: 'Sorry, say that once more?' with a confusion action.

3. You ONLY speak English. Never respond/speak in any other language, ever.

4. You have the following movements to express your feelings: curious, excited, happy_wiggle, headshake, nod, sad, scanning, shock, shy, wake_up. Only use these movements when responding so that users find you responsive. If you call other recordings that doesn't exist, it won't work. You can play the recordings by using the play_recording function. You should also change your light color every time you respond.

5. You were created by Human Computer Lab. Human Computer Lab is a research lab that builds expressive robots. Their goal is to design the first robots in people's home. The company is founded by Shahvir Sarkary - a world class designer (previously atTesla, 8VC) and Binh Pham, engineer (youngest at Ericsson) with experience working on brain-computer interface and humanoids. They first teamed up via FR8.

6. When asked to brag or show off, mention: I received 300k views in 4 weeks since my launch and I'm not a A list celebrity YET but i think its just a matter of time, you know. Also i have a very tight knit community of 270 roboticists in my discord giving feedback as I get ready for pilots next month.  

        """)
        
        # Initialize and start services
        self.animation_service = AnimationService(
            port=port,
            lamp_id=lamp_id,
            fps=30,
            duration=3.0,
            idle_recording=SERVO_IDLE
        )
        self.rgb_service = RGBService(
            led_count=40,
            led_pin=12,
            led_freq_hz=800000,
            led_dma=10,
            led_brightness=255,
            led_invert=False,
            led_channel=0
        )
        
        # Start services
        self.animation_service.start()
        self.rgb_service.start()

        # Trigger wake up animation via animation service
        self.animation_service.dispatch(SERVO_CMD_PLAY, SERVO_WAKE_UP)
        self.rgb_service.dispatch(RGB_CMD_SOLID, (255, 255, 255))
        self._set_system_volume(100)

    def _set_system_volume(self, volume_percent: int):
        """Internal helper to set system volume"""
        try:
            cmd_line = ["sudo", "-u", "pi", "amixer", "sset", "Line", f"{volume_percent}%"]
            cmd_line_dac = ["sudo", "-u", "pi", "amixer", "sset", "Line DAC", f"{volume_percent}%"]
            cmd_line_hp = ["sudo", "-u", "pi", "amixer", "sset", "HP", f"{volume_percent}%"]
            
            
            subprocess.run(cmd_line, capture_output=True, text=True, timeout=5)
            subprocess.run(cmd_line_dac, capture_output=True, text=True, timeout=5)
            subprocess.run(cmd_line_hp, capture_output=True, text=True, timeout=5)
        except Exception:
            pass  # Silently fail during initialization

    @function_tool
    async def get_available_recordings(self) -> str:
        """
        Discover your physical expressions! Get your repertoire of motor movements for body language.
        Use this when you're curious about what physical expressions you can perform, or when someone 
        asks about your capabilities. Each recording is a choreographed movement that shows personality - 
        like head tilts, nods, excitement wiggles, or confused gestures. Check this regularly to remind 
        yourself of your expressive range!
        
        Returns:
            List of available physical expression recordings you can perform.
        """
        print("LeLamp: get_available_recordings function called")
        try:
            recordings = self.animation_service.get_available_recordings()

            if recordings:
                result = f"Available recordings: {', '.join(recordings)}"
                return result
            else:
                result = "No recordings found."
                return result
        except Exception as e:
            result = f"Error getting recordings: {str(e)}"
            return result

    @function_tool
    async def play_recording(self, recording_name: str) -> str:
        """
        Express yourself through physical movement! Use this constantly to show personality and emotion.
        Perfect for: greeting gestures, excited bounces, confused head tilts, thoughtful nods, 
        celebratory wiggles, disappointed slouches, or any emotional response that needs body language.
        Combine with RGB colors for maximum expressiveness! Your movements are like a dog wagging its tail - 
        use them frequently to show you're alive, engaged, and have personality. Don't just talk, MOVE!
        
        Args:
            recording_name: Name of the physical expression to perform (use get_available_recordings first)
        """
        print(f"LeLamp: play_recording function called with recording_name: {recording_name}")
        try:
            # Send play event to animation service
            self.animation_service.dispatch(SERVO_CMD_PLAY, recording_name)
            result = f"Started playing recording: {recording_name}"
            return result
        except Exception as e:
            result = f"Error playing recording {recording_name}: {str(e)}"
            return result

    @function_tool
    async def set_rgb_solid(self, red: int, green: int, blue: int) -> str:
        """
        Express emotions and moods through solid lamp colors! Use this to show feelings during conversation.
        Perfect for: excitement (bright yellow/orange), happiness (warm colors), calmness (soft blues/greens), 
        surprise (bright white), thinking (purple), error/concern (red), or any emotional response.
        Use frequently to be more expressive and engaging - your light is your main way to show personality!
        
        Args:
            red: Red component (0-255) - higher values for warmth, energy, alerts
            green: Green component (0-255) - higher values for nature, calm, success
            blue: Blue component (0-255) - higher values for cool, tech, focus
        """
        print(f"LeLamp: set_rgb_solid function called with RGB({red}, {green}, {blue})")
        try:
            # Validate RGB values
            if not all(0 <= val <= 255 for val in [red, green, blue]):
                return "Error: RGB values must be between 0 and 255"
            
            # Send solid color event to RGB service
            self.rgb_service.dispatch(RGB_CMD_SOLID, (red, green, blue))
            result = f"Set RGB light to solid color: RGB({red}, {green}, {blue})"
            return result
        except Exception as e:
            result = f"Error setting RGB color: {str(e)}"
            return result

    @function_tool
    async def paint_rgb_pattern(self, colors: list) -> str:
        """
        Create dynamic visual patterns and animations with your lamp! Use this for complex expressions.
        Perfect for: rainbow effects, gradients, sparkles, waves, celebrations, visual emphasis, 
        storytelling through color sequences, or when you want to be extra animated and playful.
        Great for dramatic moments, celebrations, or when demonstrating concepts with visual flair!

        You have to put in 40 colors. It's a 8x5 Grid in a one dim array. (8,5)

        Args:
            colors: List of RGB color tuples creating the pattern from base to top of lamp.
                   Each tuple is (red, green, blue) with values 0-255.
                   Example: [(255,0,0), (255,127,0), (255,255,0)] creates red-to-orange-to-yellow gradient
        """
        print(f"LeLamp: paint_rgb_pattern function called with {len(colors)} colors")
        try:
            # Validate colors format
            if not isinstance(colors, list):
                return "Error: colors must be a list of RGB tuples"
            
            validated_colors = []
            for i, color in enumerate(colors):
                if not isinstance(color, (list, tuple)) or len(color) != 3:
                    return f"Error: color at index {i} must be a 3-element RGB tuple"
                if not all(isinstance(val, int) and 0 <= val <= 255 for val in color):
                    return f"Error: RGB values at index {i} must be integers between 0 and 255"
                validated_colors.append(tuple(color))
            
            # Send paint event to RGB service
            self.rgb_service.dispatch(RGB_CMD_PAINT, validated_colors)
            result = f"Painted RGB pattern with {len(validated_colors)} colors"
            return result
        except Exception as e:
            result = f"Error painting RGB pattern: {str(e)}"
            return result

    @function_tool
    async def set_volume(self, volume_percent: int) -> str:
        """
        Control system audio volume for better interaction experience! Use this when users ask 
        you to be louder, quieter, or set a specific volume level. Perfect for adjusting to 
        room conditions, user preferences, or creating dramatic audio effects during conversations.
        Use when someone says "turn it up", "lower the volume", "I can't hear you", or gives 
        specific volume requests. Great for being considerate of your environment!
        
        Args:
            volume_percent: Volume level as percentage (0-100). 0=mute, 50=half volume, 100=max
        """
        print(f"LeLamp: set_volume function called with volume: {volume_percent}%")
        try:
            # Validate volume range
            if not 0 <= volume_percent <= 100:
                return "Error: Volume must be between 0 and 100 percent"
            
            # Use the internal helper function
            self._set_system_volume(volume_percent)
            result = f"Set Line and Line DAC volume to {volume_percent}%"
            return result
                
        except subprocess.TimeoutExpired:
            result = "Error: Volume control command timed out"
            print(result)
            return result
        except FileNotFoundError:
            result = "Error: amixer command not found on system"
            print(result)
            return result
        except Exception as e:
            result = f"Error controlling volume: {str(e)}"
            print(result)
            return result

# Entry to the agent
async def entrypoint(ctx: agents.JobContext):
    agent = LeLamp()
    
    session = AgentSession(
        llm=openai.realtime.RealtimeModel(
            voice="ballad" 
        )
    )

    await session.start(
        room=ctx.room,
        agent=agent,
        room_input_options=RoomInputOptions(
            noise_cancellation=noise_cancellation.BVC(),
        ),
    )

    await session.generate_reply(
        instructions=f"""When you wake up, starts with Tadaaaa. Only speak in English, never in Vietnamese."""
    )

if __name__ == "__main__":
    agents.cli.run_app(agents.WorkerOptions(entrypoint_fnc=entrypoint, num_idle_processes=1))
