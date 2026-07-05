inspired by the Instagram reel

game state : live -> puzzle -> done 

https://www.instagram.com/reel/DZ1YQ4HtCHy

Debian/Ubuntu:

 <code>sudo apt install libc6-dev libgl1-mesa-dev libxcursor-dev libxi-dev libxinerama-dev libxrandr-dev libxxf86vm-dev libasound2-dev pkg-config</code>
 
 Clone and build TensorFlow Lite (minimal)

`git clone https://github.com/tensorflow/tensorflow.git --depth 1 --branch v2.15.0`

`cd tensorflow/tensorflow/lite`

`mkdir build && cd build`

`cmake .. -DTFLITE_ENABLE_XNNPACK=ON -DCMAKE_BUILD_TYPE=Release`

`make -j$(nproc) tflite`

 Install headers and lib
`sudo make install`
`sudo ldconfig`
