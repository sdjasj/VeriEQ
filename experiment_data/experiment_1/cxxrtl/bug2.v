`timescale 1ns/1ps
module a (in2, out2);

input wire in2;
output wire out2;

assign out2 = (7'b11100 / ({1'b1, in2}));

endmodule