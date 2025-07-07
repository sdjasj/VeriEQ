`timescale 1ns/1ps
module top (in3, in4, out10);
    input wire  [7:7] in3;
    input wire [23:16] in4;
    wire  [29:29] wire_2;
    output wire  [28:14] out10;

    assign wire_2 = (((in3) % 4'o16) ? {{4{14'b010111101}}} : (in4[18:18] >> 8'b0001));
    assign out10 = wire_2;
endmodule