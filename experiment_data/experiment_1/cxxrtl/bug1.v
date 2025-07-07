`timescale 1ns/1ps
module cqsnqdxwkr (in2, out0);
reg signed [9:0] reg_7;
input wire in2;
output wire [12:0] out0;

    initial begin
        reg_7 = 848;
    end
  
    assign out0 = reg_7 >> in2;
endmodule